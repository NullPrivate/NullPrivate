import React from 'react';
import { render, unmountComponentAtNode } from 'react-dom';
import { act, Simulate } from 'react-dom/test-utils';
import { Provider } from 'react-redux';
import { createStore } from 'redux';
import { expect, it, vi } from 'vitest';

import { Auth } from '../install/Setup/Auth';

vi.mock('react-i18next', () => ({
    initReactI18next: { type: '3rdParty', init: () => {} },
    useTranslation: () => ({ t: (key: string) => key }),
    Trans: ({ children }: { children: React.ReactNode }) => children,
}));

it('填写有效安装账号后启用下一步并提交账号信息', async () => {
    const container = document.createElement('div');
    document.body.appendChild(container);
    const onAuthSubmit = vi.fn();
    const store = createStore(() => ({ install: { step: 3, processingSubmit: false } }));

    try {
        await act(async () => {
            render(
                <Provider store={store}>
                    <Auth onAuthSubmit={onAuthSubmit} />
                </Provider>,
                container,
            );
        });
        const next = container.querySelector('[data-testid="install_next"]') as HTMLButtonElement;
        expect(next.disabled).toBe(true);

        const values = { username: 'admin', password: 'secure-password', confirm_password: 'secure-password' };
        const fillField = async (name: string, value: string) => {
            const input = container.querySelector(`input[data-testid="install_${name}"]`) as HTMLInputElement;
            await act(async () => {
                input.value = value;
                Simulate.change(input);
            });
            await act(async () => {
                Simulate.blur(input);
            });
        };
        await fillField('username', values.username);
        await fillField('password', values.password);
        await fillField('confirm_password', values.confirm_password);

        expect(next.disabled).toBe(false);
        await act(async () => {
            Simulate.submit(container.querySelector('form'));
        });
        expect(onAuthSubmit).toHaveBeenCalledWith(values, expect.anything());
    } finally {
        act(() => {
            unmountComponentAtNode(container);
        });
        container.remove();
    }
});
