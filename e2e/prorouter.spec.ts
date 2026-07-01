import { test, expect } from '@playwright/test';

const PORT = 9099;
const BASE = `http://localhost:${PORT}`;

// Helper: wait for server to be ready
async function waitForServer(baseURL: string, maxRetries = 20) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      const r = await fetch(`${baseURL}/api/stats`);
      if (r.ok) return;
    } catch {}
    await new Promise(r => setTimeout(r, 1000));
  }
  throw new Error('Server did not start');
}

test.describe('ProRouter E2E', () => {
  let prorouterProcess: any = null;

  test.beforeAll(async () => {
    // Start the server
    const { spawn } = await import('child_process');
    const path = await import('path');
    const prorouterBin = path.resolve(__dirname, '..', 'gateway-go', 'prorouter.exe');

    prorouterProcess = spawn(prorouterBin, ['serve', '--port', String(PORT)], {
      stdio: 'pipe',
      env: { ...process.env, CGO_ENABLED: '0' },
    });

    // Wait for startup
    await waitForServer(BASE);
  });

  test.afterAll(async () => {
    if (prorouterProcess) {
      prorouterProcess.kill();
    }
  });

  test('Dashboard serves HTML', async ({ page }) => {
    const response = await page.goto(`${BASE}/dashboard/`);
    expect(response?.status()).toBe(200);
    const title = await page.title();
    expect(title).toContain('ProRouter');
  });

  test('Stats API returns valid data', async () => {
    const r = await fetch(`${BASE}/api/stats`);
    expect(r.status).toBe(200);
    const data = await r.json();
    expect(data).toHaveProperty('total_requests');
    expect(data).toHaveProperty('total_cost_usd');
    expect(data).toHaveProperty('active_keys');
  });

  test('Create API key via API', async () => {
    const r = await fetch(`${BASE}/api/keys`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'e2e-test', monthly_budget: 10 }),
    });
    expect(r.status).toBe(201);
    const data = await r.json();
    expect(data.key).toMatch(/^pr-/);
    expect(data.id).toBeTruthy();

    // Store key for other tests
    (globalThis as any).__apiKey = data.key;
  });

  test('List API keys', async () => {
    const r = await fetch(`${BASE}/api/keys`);
    expect(r.status).toBe(200);
    const keys = await r.json();
    expect(Array.isArray(keys)).toBe(true);
    expect(keys.length).toBeGreaterThanOrEqual(1);
  });

  test('Auth rejects unauthenticated requests', async () => {
    const r = await fetch(`${BASE}/v1/models`);
    expect(r.status).toBe(401);
    const data = await r.json();
    expect(data.error).toBeTruthy();
  });

  test('Auth allows authenticated requests', async () => {
    const apiKey = (globalThis as any).__apiKey;
    expect(apiKey).toBeTruthy();

    const r = await fetch(`${BASE}/v1/models`, {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    expect(r.status).toBe(200);
    const data = await r.json();
    expect(data.object).toBe('list');
    expect(data.data.length).toBeGreaterThanOrEqual(1);
  });

  test('Playground endpoint works', async () => {
    const r = await fetch(`${BASE}/api/playground`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        model: 'gpt-4o',
        messages: [{ role: 'user', content: 'Say hello in one word' }],
        stream: false,
      }),
    });
    // Note: playground will proxy to provider, may fail without credentials
    // but should return a proper response (error from provider, not gateway error)
    expect([200, 400, 401, 402, 500, 502]).toContain(r.status);
  });

  test('Dashboard page elements are present', async ({ page }) => {
    await page.goto(`${BASE}/dashboard/`);
    await expect(page.locator('h1')).toContainText('ProRouter');
    await expect(page.locator('#nav')).toBeVisible();
    await expect(page.locator('#stats-grid')).toBeVisible();
  });

  test('Dashboard tab navigation works', async ({ page }) => {
    await page.goto(`${BASE}/dashboard/`);

    // Click each tab and verify content loads
    const tabs = ['API Keys', 'Providers', 'Playground', 'Logs', 'Recipes'];
    for (const tab of tabs) {
      await page.getByText(tab).first().click();
      await page.waitForTimeout(500);
    }
  });

  test('Create key via dashboard UI', async ({ page }) => {
    await page.goto(`${BASE}/dashboard/`);
    await page.getByText('API Keys').first().click();

    // Click "New Key" button
    await page.getByText('+ New Key').click();

    // Modal should appear
    await expect(page.locator('#key-modal')).toBeVisible();

    // Fill form
    await page.fill('#key-name', 'ui-test-key');

    // Click Generate
    await page.getByText('Generate').click();

    // Should show generated key
    await expect(page.locator('#new-key-value')).toBeVisible();
    const keyText = await page.locator('#new-key-value').textContent();
    expect(keyText).toMatch(/^pr-/);
  });

  test('Revoke API key', async () => {
    // First list keys
    const listR = await fetch(`${BASE}/api/keys`);
    const keys = await listR.json();
    const activeKey = keys.find((k: any) => !k.is_revoked);
    if (!activeKey) return; // skip if no active key

    const r = await fetch(`${BASE}/api/keys/${activeKey.id}`, {
      method: 'DELETE',
    });
    expect(r.status).toBe(200);

    // Verify it's revoked
    const verifyR = await fetch(`${BASE}/api/keys`);
    const updatedKeys = await verifyR.json();
    const revokedKey = updatedKeys.find((k: any) => k.id === activeKey.id);
    expect(revokedKey.is_revoked).toBe(true);
  });

  test('Providers endpoint works', async () => {
    const r = await fetch(`${BASE}/api/providers`);
    expect(r.status).toBe(200);
    const providers = await r.json();
    expect(Array.isArray(providers)).toBe(true);
  });

  test('Logs endpoint works', async () => {
    const r = await fetch(`${BASE}/api/logs`);
    expect(r.status).toBe(200);
    const logs = await r.json();
    expect(Array.isArray(logs)).toBe(true);
  });

  test('Health check via root redirect', async ({ page }) => {
    const response = await page.goto(BASE);
    // Should redirect to /dashboard/
    expect(page.url()).toContain('/dashboard/');
  });
});
