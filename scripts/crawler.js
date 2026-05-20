#!/usr/bin/env node
/**
 * Price Crawler using Playwright
 * Fetches real prices from JD/Taobao using saved cookies
 *
 * Usage: node crawler.js <url> [source]
 * Example: node crawler.js https://item.jd.com/100009077449.html jd
 */

const { chromium } = require('/tmp/node_modules/playwright');
const fs = require('fs');
const path = require('path');

const COOKIE_DIR = path.join(__dirname, '..', 'cookies');
const CHROMIUM_PATH = process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH || '/usr/bin/chromium';
const WAIT_TIME = 4000;

async function fetchPrice(url, source) {
  const browser = await chromium.launch({
    executablePath: CHROMIUM_PATH,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage', '--disable-web-security']
  });

  let context;
  try {
    context = await browser.newContext({
      userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36'
    });

    const cookieFile = path.join(COOKIE_DIR, `${source}_cookies.json`);
    if (fs.existsSync(cookieFile)) {
      const cookies = JSON.parse(fs.readFileSync(cookieFile, 'utf8'));
      await context.addCookies(cookies);
    }

    const page = await context.newPage();

    await page.setExtraHTTPHeaders({
      'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8',
      'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8'
    });

    // Listen for console messages
    page.on('console', msg => {
      if (msg.type() === 'error') console.error('BROWSER ERROR:', msg.text());
    });

    await page.goto(url, { timeout: 20000, waitUntil: 'networkidle' });
    await page.waitForTimeout(WAIT_TIME);

    let result;
    if (source === 'jd' || url.includes('jd.com')) {
      result = await extractJDPrice(page, url);
    } else if (source === 'taobao' || url.includes('taobao.com') || url.includes('tmall.com')) {
      result = await extractTaobaoPrice(page, url);
    } else {
      result = await extractGenericPrice(page, url);
    }

    await browser.close();
    return result;

  } catch (err) {
    if (context) await context.close().catch(() => {});
    await browser.close();
    throw err;
  }
}

async function extractJDPrice(page, url) {
  // JD uses many different price elements depending on product type
  // Try to get final URL after any redirects
  const finalUrl = page.url();

  const data = await page.evaluate(() => {
    // Get current page title to check if we got the right page
    const pageTitle = document.title || '';

    // Price selectors for JD - many products have different structures
    const priceSelectors = [
      // Standard product page
      '.J-price',
      '.p-price',
      '.summary-price',
      '#jd-price',
      // New JD layout
      '[class*="price-current"]',
      '[class*="priceTag"]',
      '.price-item',
      // Competitor/new layout
      '[data-price]',
      '.product-price',
      '.goods-price',
      // Watch/accessories layout
      '[class*="current-price"]',
      '[class*="sale-price"]',
      '.price-info .p-price',
      // Try any element with ¥ or ￥
    ];

    let priceText = '';
    let priceEl = null;

    for (const sel of priceSelectors) {
      const el = document.querySelector(sel);
      if (el && el.textContent.trim()) {
        const txt = el.textContent.trim();
        // Check if it looks like a price
        if (/[¥￥]/.test(txt) || /[0-9]+\.[0-9]{2}/.test(txt)) {
          priceEl = el;
          priceText = txt;
          break;
        }
      }
    }

    // Fallback: search for ¥ or ￥ in page
    if (!priceText) {
      const body = document.body.innerText;
      const priceMatches = body.match(/[¥￥]\s*([0-9,]+\.?[0-9]*)/g);
      if (priceMatches && priceMatches.length > 0) {
        priceText = priceMatches[0];
      }
      // Also try to find J_* price elements
      const jPriceEls = document.querySelectorAll('[class*="J-p"]');
      for (const el of jPriceEls) {
        const txt = el.textContent.trim();
        if (/[0-9]/.test(txt)) {
          priceText = txt;
          break;
        }
      }
    }

    // Parse numeric price
    let price = 0;
    if (priceText) {
      const cleanPrice = priceText.replace(/[¥￥,，\s]/g, '');
      const priceMatch = cleanPrice.match(/([0-9]+\.[0-9]{1,2})/);
      if (priceMatch) {
        price = parseFloat(priceMatch[1]);
      } else {
        const intMatch = cleanPrice.match(/([0-9]+)/);
        if (intMatch) price = parseFloat(intMatch[1]);
      }
    }

    // If still 0, try to find any price-like number in the page
    if (price === 0) {
      const body = document.body.innerText;
      const allPrices = body.match(/([0-9]{2,}\.[0-9]{2})/g);
      if (allPrices && allPrices.length > 0) {
        // Filter out unlikely prices (too high or too low)
        const validPrices = allPrices.map(p => parseFloat(p)).filter(p => p > 1 && p < 100000);
        if (validPrices.length > 0) {
          // Prefer prices that appear near "price" or "¥" text
          price = validPrices[0];
        }
      }
    }

    // Get title
    const titleEl = document.querySelector('.sku-name') ||
                    document.querySelector('[class*="product-title"]') ||
                    document.querySelector('[class*="item-title"]') ||
                    document.querySelector('title');
    let title = titleEl ? titleEl.textContent.trim().replace(/\s+/g, ' ').substring(0, 200) : '';

    // If title looks like JD homepage, we got redirected
    if (title.includes('京东') && title.length < 30) {
      title = '';
    }

    // Get image
    const imgEl = document.querySelector('#spec-img') ||
                  document.querySelector('.main-img img') ||
                  document.querySelector('[class*="product-img"] img') ||
                  document.querySelector('[class*="item-img"] img') ||
                  document.querySelector('[class*="gallery"] img');
    let imageUrl = '';
    if (imgEl) {
      imageUrl = imgEl.src || imgEl.getAttribute('data-src') || imgEl.getAttribute('data-lazy-img') || '';
    }

    return {
      name: title,
      price: price,
      imageUrl: imageUrl,
      source: 'jd',
      rawPrice: priceText,
      pageTitle: pageTitle
    };
  });

  return {
    name: data.name || 'JD Product',
    price: data.price,
    imageUrl: data.imageUrl,
    source: 'jd',
    productUrl: url
  };
}

async function extractTaobaoPrice(page, url) {
  await page.waitForTimeout(3000);

  const data = await page.evaluate(() => {
    const priceSelectors = [
      '.price .original',
      '.price',
      '#J_StrPrice',
      '[class*="price"]',
      '.original-price',
      '[class*="priceTag"]',
    ];

    let priceText = '';
    for (const sel of priceSelectors) {
      const el = document.querySelector(sel);
      if (el && el.textContent.trim()) {
        priceText = el.textContent.trim();
        break;
      }
    }

    let price = 0;
    if (priceText) {
      const clean = priceText.replace(/[¥￥,，\s]/g, '');
      const m = clean.match(/([0-9]+\.?[0-9]*)/);
      if (m) price = parseFloat(m[1]);
    }

    const titleEl = document.querySelector('.main-info .title') ||
                    document.querySelector('title');
    const title = titleEl ? titleEl.textContent.trim().replace(/\s+/g, ' ').substring(0, 200) : 'Taobao Product';

    const imgEl = document.querySelector('#J_ImgBooth') ||
                  document.querySelector('.tb-booth img');
    const imageUrl = imgEl ? (imgEl.src || imgEl.getAttribute('data-src') || '') : '';

    return { name: title, price, imageUrl, rawPrice: priceText };
  });

  return {
    name: data.name,
    price: data.price,
    imageUrl: data.imageUrl,
    source: 'taobao',
    productUrl: url
  };
}

async function extractGenericPrice(page, url) {
  const data = await page.evaluate(() => {
    const bodyText = document.body.innerText;

    const pricePatterns = [
      /[¥￥]\s*([0-9,]+\.[0-9]{2})/,
      /price[^0-9]*([0-9]+\.[0-9]{2})/i,
      /([0-9]{2,}\.[0-9]{2})/,
    ];

    let price = 0;
    for (const pat of pricePatterns) {
      const m = bodyText.match(pat);
      if (m) { price = parseFloat(m[1]); break; }
    }

    const title = document.querySelector('title')?.textContent?.trim() || 'Product';

    return { name: title.substring(0, 200), price };
  });

  return {
    name: data.name,
    price: data.price,
    imageUrl: '',
    source: 'generic',
    productUrl: url
  };
}

async function main() {
  const url = process.argv[2];
  const source = process.argv[3] || 'jd';

  if (!url) {
    console.error('Usage: node crawler.js <url> [source]');
    process.exit(1);
  }

  try {
    const result = await fetchPrice(url, source);
    console.log(JSON.stringify({ success: true, data: result }));
  } catch (err) {
    console.error(JSON.stringify({ success: false, error: err.message }));
    process.exit(1);
  }
}

main();
