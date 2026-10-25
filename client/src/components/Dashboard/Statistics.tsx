import React from 'react';

import { Link } from 'react-router-dom';
import { withTranslation, Trans } from 'react-i18next';

import StatsCard from './StatsCard';

import { getPercent, normalizeHistory } from '../../helpers/helpers';
import { RESPONSE_FILTER } from '../../helpers/constants';

const getNormalizedHistory = (data: any, interval: any, id: any) => [{ data: normalizeHistory(data), id }];

interface StatisticsProps {
    interval: number;
    dnsQueries: number[];
    blockedFiltering: unknown[];
    numDnsQueries: number;
    numBlockedFiltering: number;
    refreshButton: React.ReactNode;
}

const Statistics = ({
    interval,
    dnsQueries,
    blockedFiltering,
    numDnsQueries,
    numBlockedFiltering,
}: StatisticsProps) => (
    <div className="row">
        <div className="col-sm-6">
            <StatsCard
                total={numDnsQueries}
                lineData={getNormalizedHistory(dnsQueries, interval, 'dnsQuery')}
                title={
                    <Link to="logs">
                        <Trans>dns_query</Trans>
                    </Link>
                }
                color="blue"
            />
        </div>

        <div className="col-sm-6">
            <StatsCard
                total={numBlockedFiltering}
                lineData={getNormalizedHistory(blockedFiltering, interval, 'blockedFiltering')}
                percent={getPercent(numDnsQueries, numBlockedFiltering)}
                title={
                    <Trans
                        components={[
                            <Link to={`logs?response_status=${RESPONSE_FILTER.BLOCKED.QUERY}`} key="0">
                                link
                            </Link>,
                        ]}>
                        blocked_by
                    </Trans>
                }
                color="red"
            />
        </div>
    </div>
);

export default withTranslation()(Statistics);
