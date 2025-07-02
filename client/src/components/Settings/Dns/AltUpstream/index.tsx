import React from 'react';
import { useTranslation } from 'react-i18next';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';

import Form from './Form';

import Card from '../../../ui/Card';
import { setDnsConfig } from '../../../../actions/dnsConfig';
import { RootState } from '../../../../initialState';

const AltUpstream = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const {
        upstream_alternate_dns,
        upstream_alternate_rulesets,
    } = useSelector((state: RootState) => state.dnsConfig, shallowEqual);

    const upstream_dns_file = useSelector((state: RootState) => state.dnsConfig.upstream_dns_file);

    const handleSubmit = (values: any) => {
        const {
            upstream_alternate_dns,
            upstream_alternate_rulesets,
        } = values;

        const dnsConfig = {
            upstream_alternate_rulesets,
            ...(upstream_dns_file ? null : { upstream_alternate_dns }),
        };

        dispatch(setDnsConfig(dnsConfig));
    };

    const upstreamAlternateDns = upstream_dns_file
        ? t('upstream_dns_configured_in_file', { path: upstream_dns_file })
        : upstream_alternate_dns;

    return (
        <Card title={t('upstream_alternate_dns_title')} bodyType="card-body box-body--settings">
            <div className="row">
                <div className="col">
                    <Form
                        initialValues={{
                            upstream_alternate_dns: upstreamAlternateDns,
                            upstream_alternate_rulesets,
                        }}
                        onSubmit={handleSubmit}
                    />
                </div>
            </div>
        </Card>
    );
};

export default AltUpstream;