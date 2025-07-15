import { handleActions } from 'redux-actions';

import * as actions from '../actions/services';

const services = handleActions(
    {
        [actions.getBlockedServicesRequest.toString()]: (state: any) => ({
            ...state,
            processing: true,
        }),
        [actions.getBlockedServicesFailure.toString()]: (state: any) => ({
            ...state,
            processing: false,
        }),
        [actions.getBlockedServicesSuccess.toString()]: (state, { payload }: any) => ({
            ...state,
            list: payload,
            processing: false,
        }),

        [actions.getAllBlockedServicesRequest.toString()]: (state: any) => ({
            ...state,
            processingAll: true,
        }),
        [actions.getAllBlockedServicesFailure.toString()]: (state: any) => ({
            ...state,
            processingAll: false,
        }),
        [actions.getAllBlockedServicesSuccess.toString()]: (state, { payload }: any) => ({
            ...state,
            allServices: payload.blocked_services,
            processingAll: false,
        }),

        [actions.updateBlockedServicesRequest.toString()]: (state: any) => ({
            ...state,
            processingSet: true,
        }),
        [actions.updateBlockedServicesFailure.toString()]: (state: any) => ({
            ...state,
            processingSet: false,
        }),
        [actions.updateBlockedServicesSuccess.toString()]: (state: any) => ({
            ...state,
            processingSet: false,
        }),

        [actions.reloadBlockedServicesRequest.toString()]: (state: any) => ({
            ...state,
            processingReload: true,
        }),
        [actions.reloadBlockedServicesFailure.toString()]: (state: any) => ({
            ...state,
            processingReload: false,
        }),
        [actions.reloadBlockedServicesSuccess.toString()]: (state: any) => ({
            ...state,
            processingReload: false,
        }),

        [actions.getServiceUrlsRequest.toString()]: (state: any) => ({
            ...state,
            processingGetUrls: true,
        }),
        [actions.getServiceUrlsFailure.toString()]: (state: any) => ({
            ...state,
            processingGetUrls: false,
        }),
        [actions.getServiceUrlsSuccess.toString()]: (state: any, { payload }: any) => ({
            ...state,
            serviceUrls: payload?.service_urls || [],
            processingGetUrls: false,
        }),

        [actions.updateServiceUrlsRequest.toString()]: (state: any) => ({
            ...state,
            processingSetUrls: true,
        }),
        [actions.updateServiceUrlsFailure.toString()]: (state: any) => ({
            ...state,
            processingSetUrls: false,
        }),
        [actions.updateServiceUrlsSuccess.toString()]: (state: any) => ({
            ...state,
            processingSetUrls: false,
        }),
    },
    {
        processing: true,
        processingAll: true,
        processingSet: false,
        processingReload: false,
        processingGetUrls: false,
        processingSetUrls: false,
        list: {},
        allServices: [],
        serviceUrls: [],
    },
);

export default services;
