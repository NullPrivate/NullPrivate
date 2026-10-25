import React from 'react';
import { render, unmountComponentAtNode } from 'react-dom';
import { act } from 'react-dom/test-utils';
import { Provider } from 'react-redux';
import { createStore } from 'redux';
import { MemoryRouter } from 'react-router-dom';
import { FormProvider, useForm } from 'react-hook-form';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import axios from 'axios';

import Settings from '../components/Settings';
import { MainSettings } from '../components/Settings/Clients/Form/components/MainSettings';
import { BlockedServices } from '../components/Settings/Clients/Form/components/BlockedServices';
import { ClientForm } from '../components/Settings/Clients/Form/types';
import Counters from '../components/Dashboard/Counters';
import Statistics from '../components/Dashboard/Statistics';
import { Form as LogsFilters } from '../components/Logs/Filters/Form';
import ResponseCell from '../components/Logs/Cells/ResponseCell';
import { initSettings } from '../actions';
import settingsReducer from '../reducers/settings';
import apiClient from '../api/Api';
import { FILTERED_STATUS, TIME_UNITS } from '../helpers/constants';

vi.mock('axios', () => ({ default: vi.fn() }));
vi.mock('react-i18next', () => ({
    initReactI18next: { type: '3rdParty', init: () => {} },
    useTranslation: () => ({ t: (key: string) => key }),
    Trans: ({ children }: { children: React.ReactNode }) => children,
    withTranslation: () => (Component: React.ComponentType<any>) => {
        const Translated = (props: any) => <Component {...props} t={(key: string) => key} />;
        return Translated;
    },
}));
// jsdom 使用可观察的图表数据替身，其余统计卡片与 Redux 数据流保持真实。
vi.mock('@nivo/line', () => ({
    ResponsiveLine: ({ data }: { data: { id: string }[] }) => <svg data-series={data[0].id} />,
}));

const safeSearch = { enabled: true, google: true, bing: true };
const legacySettings = { parental: { enabled: true }, safebrowsing: { enabled: true } };
const stats = {
    interval: 86400000,
    timeUnits: TIME_UNITS.HOURS,
    numDnsQueries: 100,
    numBlockedFiltering: 20,
    numReplacedSafebrowsing: 7,
    numReplacedParental: 8,
    numReplacedSafesearch: 5,
    avgProcessingTime: 2,
};
const state = {
    service_type: 'family',
    stats,
    filtering: { filters: [], whitelistFilters: [] },
    queryLogs: { isDetailed: false },
    services: { allServices: [] },
};

const ClientHarness = ({ onSubmit }: { onSubmit: (values: Partial<ClientForm>) => void }) => {
    const methods = useForm<Partial<ClientForm>>({
        defaultValues: {
            use_global_settings: false,
            filtering_enabled: true,
            safebrowsing_enabled: true,
            parental_enabled: true,
            safe_search: safeSearch,
            ignore_querylog: false,
            ignore_statistics: false,
            use_global_blocked_services: false,
            blocked_services: { youtube: false },
        },
    });
    return (
        <FormProvider {...methods}>
            <form onSubmit={methods.handleSubmit(onSubmit)}>
                <MainSettings safeSearchServices={{ google: true, bing: true }} />
                <BlockedServices services={[{ id: 'youtube', name: 'YouTube', icon_svg: '' }]} />
                <button type="submit">保存</button>
            </form>
        </FormProvider>
    );
};

const LogsHarness = () => {
    const methods = useForm({ defaultValues: { search: '', response_status: 'all' } });
    return (
        <FormProvider {...methods}>
            <LogsFilters setIsLoading={vi.fn()} />
        </FormProvider>
    );
};

describe('NullPrivate 前端保留能力集成', () => {
    let container: HTMLDivElement;

    beforeEach(() => {
        vi.mocked(axios).mockReset();
        container = document.createElement('div');
        document.body.appendChild(container);
        window.matchMedia = () =>
            ({ matches: false, addListener: vi.fn(), removeListener: vi.fn() }) as unknown as MediaQueryList;
    });

    afterEach(() => {
        act(() => {
            unmountComponentAtNode(container);
        });
        container.remove();
    });

    const mount = async (element: React.ReactElement, serviceType = 'family') => {
        const store = createStore(() => ({ ...state, service_type: serviceType }));
        await act(async () => {
            render(
                <Provider store={store}>
                    <MemoryRouter>{element}</MemoryRouter>
                </Provider>,
                container,
            );
        });
        return store;
    };

    const checkboxIds = (root: Element = container) =>
        Array.from(root.querySelectorAll('input[type="checkbox"]')).map((input) => input.getAttribute('data-testid'));

    const click = async (testId: string) => {
        await act(async () => {
            (container.querySelector(`[data-testid="${testId}"]`) as HTMLInputElement).click();
        });
    };

    it.each(['family', 'enterprise'])('%s 全局设置展示规则过滤、安全搜索和日志统计配置', async (serviceType) => {
        const toggleSetting = vi.fn();
        const setFiltersConfig = vi.fn();
        await mount(
            <Settings
                settings={{ processing: false, settingsList: { ...legacySettings, safesearch: safeSearch } }}
                filtering={{ enabled: true, interval: 24 }}
                stats={{ enabled: true, interval: 86400000, ignored: [] }}
                queryLogs={{ enabled: true, interval: 86400000, ignored: [] }}
                initSettings={vi.fn()}
                getStatsConfig={vi.fn()}
                getLogsConfig={vi.fn()}
                getFilteringStatus={vi.fn()}
                setStatsConfig={vi.fn()}
                resetStats={vi.fn()}
                setLogsConfig={vi.fn()}
                clearLogs={vi.fn()}
                setFiltersConfig={setFiltersConfig}
                toggleSetting={toggleSetting}
            />,
            serviceType,
        );
        const generalSettings = container.querySelector('.box-body--settings') as Element;
        expect(checkboxIds(generalSettings)).toEqual(['filters_enabled', 'safesearch', null, null]);
        expect(checkboxIds()).toContain('logs_enabled');
        expect(checkboxIds()).toContain('stats_config_enabled');
        await click('safesearch');
        expect(toggleSetting).toHaveBeenCalledWith('safesearch', { ...safeSearch, enabled: false });
        await click('filters_enabled');
        expect(setFiltersConfig).toHaveBeenCalledWith({ enabled: false, interval: 24 });
    });

    it.each(['family', 'enterprise'])('%s 客户端保留过滤、安全搜索、防沉迷和兼容字段', async (serviceType) => {
        const onSubmit = vi.fn();
        await mount(<ClientHarness onSubmit={onSubmit} />, serviceType);
        expect(checkboxIds()).toEqual([
            'clients_use_global_settings',
            'clients_filtering_enabled',
            'clients_safe_search',
            'clients_safe_search_google',
            'clients_safe_search_bing',
            'clients_ignore_querylog',
            'clients_ignore_statistics',
            'clients_use_global_blocked_services',
            'clients_service_youtube',
        ]);
        await click('clients_service_youtube');
        await click('clients_safe_search_bing');
        await act(async () => {
            container.querySelector('form').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
        });
        expect(onSubmit.mock.calls[0][0]).toMatchObject({
            filtering_enabled: true,
            safebrowsing_enabled: true,
            parental_enabled: true,
            safe_search: { ...safeSearch, bing: false },
            blocked_services: { youtube: true },
        });
    });

    it('初始化通过安全搜索 API 更新状态并保留已有兼容状态', async () => {
        vi.mocked(axios).mockResolvedValue({ data: safeSearch });
        let settings = { processing: false, settingsList: legacySettings };
        await initSettings()((action: any) => {
            settings = settingsReducer(settings, action);
        });
        expect(axios).toHaveBeenCalledTimes(1);
        expect(axios).toHaveBeenCalledWith(expect.objectContaining({ url: 'control/safesearch/status', method: 'GET' }));
        expect(settings).toMatchObject({
            processing: false,
            settingsList: { ...legacySettings, safesearch: safeSearch },
        });
    });

    it('仪表盘展示查询、规则拦截、安全搜索计数和两张趋势图', async () => {
        await mount(
            <>
                <Counters refreshButton={null} subtitle="最近一天" />
                <Statistics
                    interval={stats.interval}
                    dnsQueries={[70, 30]}
                    blockedFiltering={[15, 5]}
                    numDnsQueries={100}
                    numBlockedFiltering={20}
                    refreshButton={null}
                />
            </>,
        );
        expect(Array.from(container.querySelectorAll('.counters__title')).map((node) => node.textContent)).toEqual([
            'dns_query', 'blocked_by', 'enforced_save_search', 'average_processing_time',
        ]);
        expect(Array.from(container.querySelectorAll('.counters strong')).map((node) => node.textContent)).toEqual([
            '100', '20', '5', '2 ms',
        ]);
        expect(Array.from(container.querySelectorAll('[data-series]')).map((node) => node.getAttribute('data-series')))
            .toEqual(['dnsQuery', 'blockedFiltering']);
    });

    it('日志筛选展示当前完整选项并支持安全搜索筛选', async () => {
        await mount(<LogsHarness />);
        const select = container.querySelector('select');
        expect(Array.from(select.options).map((option) => option.value)).toEqual([
            'all', 'filtered', 'processed', 'blocked', 'blocked_services', 'whitelisted', 'rewritten', 'safe_search',
        ]);
        await act(async () => {
            select.value = 'safe_search';
            select.dispatchEvent(new Event('change', { bubbles: true }));
        });
        expect(select.value).toBe('safe_search');
    });

    it.each([
        [FILTERED_STATUS.FILTERED_SAFE_BROWSING, 'blocked_threats'],
        [FILTERED_STATUS.FILTERED_PARENTAL, 'blocked_adult_websites'],
        [FILTERED_STATUS.FILTERED_BLOCKED_SERVICE, 'blocked_services'],
    ])('历史日志 %s 继续展示原因文本', async (reason, label) => {
        await mount(
            <ResponseCell
                reason={reason}
                elapsedMs="1"
                originalResponse={[]}
                response={[]}
                rules={[]}
                status="NOERROR"
                upstream=""
                cached={false}
            />,
        );
        expect(container.querySelector('[role="gridcell"]').textContent).toContain(label);
    });

    it('兼容 API 方法继续使用原有状态端点', async () => {
        vi.mocked(axios).mockResolvedValue({ data: { enabled: true } });
        await expect(apiClient.getSafebrowsingStatus()).resolves.toEqual({ enabled: true });
        await expect(apiClient.getParentalStatus()).resolves.toEqual({ enabled: true });
        expect(axios).toHaveBeenCalledTimes(2);
        expect(axios).toHaveBeenNthCalledWith(1, expect.objectContaining({ url: 'control/safebrowsing/status' }));
        expect(axios).toHaveBeenNthCalledWith(2, expect.objectContaining({ url: 'control/parental/status' }));
    });
});
