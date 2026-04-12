import { beforeEach, describe, expect, it, vi } from 'vitest';
import axios from 'axios';

import apiClient from '../api/Api';

vi.mock('axios', () => ({
    default: vi.fn(),
}));

const mockedAxios = vi.mocked(axios);

describe('Api.makeRequest', () => {
    beforeEach(() => {
        mockedAxios.mockReset();
    });

    it('adds a default JSON content type for requests with a body', async () => {
        mockedAxios.mockResolvedValue({ data: { ok: true } });

        await expect(apiClient.makeRequest('profile/update', 'POST', { data: { theme: 'dark' } })).resolves.toEqual({
            ok: true,
        });

        expect(mockedAxios).toHaveBeenCalledWith({
            url: 'control/profile/update',
            method: 'POST',
            data: { theme: 'dark' },
            headers: {
                'Content-Type': 'application/json',
            },
        });
    });

    it('preserves an explicit content type when one is already set', async () => {
        mockedAxios.mockResolvedValue({ data: { ok: true } });

        await apiClient.makeRequest('dns_config', 'POST', {
            data: '{"enabled":true}',
            headers: {
                'Content-Type': 'application/yaml',
                Authorization: 'Bearer token',
            },
        });

        expect(mockedAxios).toHaveBeenCalledWith({
            url: 'control/dns_config',
            method: 'POST',
            data: '{"enabled":true}',
            headers: {
                'Content-Type': 'application/yaml',
                Authorization: 'Bearer token',
            },
        });
    });

    it('includes upstream response details in thrown errors', async () => {
        mockedAxios.mockRejectedValue({
            response: {
                data: 'forbidden',
                status: 500,
            },
        });

        await expect(apiClient.makeRequest('status', 'GET')).rejects.toThrow('control/status | forbidden | 500');
    });

    it('includes transport error messages in thrown errors', async () => {
        mockedAxios.mockRejectedValue(new Error('socket hang up'));

        await expect(apiClient.makeRequest('status', 'GET')).rejects.toThrow('control/status | socket hang up');
    });
});
