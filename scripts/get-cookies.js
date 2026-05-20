#!/usr/bin/env node
const { chromium } = require('/tmp/node_modules/playwright');
const fs = require('fs');
const path = require('path');

const COOKIE_DIR = path.join(__dirname, '..', 'cookies');

async function saveCookies(siteName, url) {
  console.log(`\n========== ${siteName} Cookie Saver ==========`);
  console.log(`Opening browser for ${siteName}...`);
  console.log('Please LOG IN manually, then press Enter in terminal to save cookies.');
  console.log(`Target URL: ${url}`);
  
  const browser = await chromium.launch({ 
    executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH || '/usr/bin/chromium',
    headless: false,
    args: ['--no-sandbox', '--disable-setuid-sandbox']
  });
  
  const context = await browser.newContext();
  const page = await context.newPage();
  
  await page.goto(url, { waitUntil: 'networkidle', timeout: 60000 });
  
  // Wait for user to log in
  console.log('\n>>> Waiting for manual login...');
  console.log('>>> After you finish logging in, switch to this terminal and press Enter <<<\n');
  
  await new Promise(resolve => {
    process.stdin.once('data', () => resolve());
  });
  
  // Save cookies
  const cookies = await context.cookies();
  const cookieFile = path.join(COOKIE_DIR, `${siteName.toLowerCase().replace(/\//g, '_')}_cookies.json`);
  
  fs.writeFileSync(cookieFile, JSON.stringify(cookies, null, 2));
  console.log(`\n[OK] Saved ${cookies.length} cookies to: ${cookieFile}`);
  
  // Show some cookie info
  const domain = cookies.find(c => c.name === 'thor') || cookies[0];
  if (domain) console.log(`[OK] Domain: ${domain.domain}`);
  
  await browser.close();
  console.log(`[OK] ${siteName} cookie capture complete!\n`);
}

async function main() {
  // Ensure cookie directory exists
  if (!fs.existsSync(COOKIE_DIR)) {
    fs.mkdirSync(COOKIE_DIR, { recursive: true });
  }
  
  await saveCookies('JD', 'https://www.jd.com');
  await saveCookies('Taobao', 'https://www.taobao.com');
  
  console.log('========== All Done ==========');
  console.log('Cookies saved. You can now use them in the crawler.');
}

main().catch(e => {
  console.error('Error:', e.message);
  process.exit(1);
});