import { test, expect } from '@playwright/test';
import { ADMIN_USERNAME, ADMIN_PASSWORD } from '../constants';

test.describe('General Settings', () => {
    test.beforeEach(async ({ page }) => {
        await page.goto('/login.html');
        await page.getByTestId('username').click();
        await page.getByTestId('username').fill(ADMIN_USERNAME);
        await page.getByTestId('password').click();
        await page.getByTestId('password').fill(ADMIN_PASSWORD);
        await page.keyboard.press('Tab');
        await page.getByTestId('sign_in').click();
        await page.waitForURL((url) => !url.href.endsWith('/login.html'));
    });

    test('展示规则过滤和安全搜索入口', async ({ page }) => {
        await page.goto('/#settings');
        await expect(page.getByTestId('filters_enabled')).toBeVisible();
        await expect(page.getByTestId('filters_interval')).toBeVisible();
        await expect(page.getByTestId('safesearch')).toBeVisible();
    });

    test('保留查询日志和统计配置入口', async ({ page }) => {
        await page.goto('/#settings');
        await expect(page.getByTestId('logs_enabled')).toBeVisible();
        await expect(page.getByTestId('stats_config_enabled')).toBeVisible();
    });

    test('should toggle safe search feature', async ({ page }) => {
        await page.goto('/#settings');

        const safeSearch = page.getByTestId('safesearch');
        const safeSearchLabel = await safeSearch.locator('xpath=following-sibling::*[1]');

        const initialState = await safeSearch.isChecked();

        await safeSearchLabel.click();

        await expect(safeSearch).not.toBeChecked({ checked: initialState });

        await safeSearchLabel.click();

        await expect(safeSearch).toBeChecked({ checked: initialState });
    });
});
