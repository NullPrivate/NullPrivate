import React from 'react';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { render, unmountComponentAtNode } from 'react-dom';
import { act } from 'react-dom/test-utils';
import { Provider } from 'react-redux';
import { createStore } from 'redux';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import i18n from 'i18next';

import StatsCard from '../components/Dashboard/StatsCard';

// jsdom 用确定的 SVG 路径代替尺寸测量，真实卡片、Redux 和生产 CSS 共同参与验证。
vi.mock('@nivo/line', () => ({
    ResponsiveLine: ({ data }: { data: { id: string; data: { x: number; y: number }[] }[] }) => {
        const start = data[0].data[0].y;
        return (
            <svg width="400" height="64" data-series={data[0].id}>
                <path d={`M0,${start}L200,4L400,60L400,64L0,64Z`} fill="blue" />
                <path d={`M0,${start}L200,4L400,60`} fill="none" />
            </svg>
        );
    },
}));

describe('统计卡片曲线布局集成', () => {
    let container: HTMLDivElement;
    let style: HTMLStyleElement;

    beforeEach(async () => {
        await i18n.init({ lng: 'zh-cn', fallbackLng: 'en', resources: {} });
        container = document.createElement('div');
        document.body.appendChild(container);
        style = document.createElement('style');
        style.textContent = readFileSync(resolve('src/components/ui/Line.css'), 'utf8');
        document.head.appendChild(style);
    });

    afterEach(() => {
        unmountComponentAtNode(container);
        container.remove();
        style.remove();
    });

    it.each([32, 32.352, 44.871])('起点为 %s 的曲线与填充保持图表原始坐标', (start) => {
        const store = createStore(() => ({ stats: { interval: 86400000, timeUnits: 'hours' } }));
        act(() => {
            render(
                <Provider store={store}>
                    <StatsCard
                        total={1000}
                        title={<span>DNS 查询</span>}
                        color="blue"
                        lineData={[{ id: 'dns', data: [{ x: 0, y: start }] }]}
                    />
                </Provider>,
                container,
            );
        });
        const paths = container.querySelectorAll('.card-chart-bg path');
        expect(paths).toHaveLength(2);
        paths.forEach((path) => {
            expect(path.getAttribute('d')).toMatch(new RegExp(`^M0,${start}`));
            expect(['', 'none']).toContain(getComputedStyle(path).transform);
        });
        expect(container.textContent).toContain('DNS 查询');
    });
});