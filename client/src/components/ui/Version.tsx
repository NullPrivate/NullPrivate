import React from 'react';
import { Trans } from 'react-i18next';
import { shallowEqual, useSelector } from 'react-redux';

import './Version.css';
import { RootState } from '../../initialState';

const Version = () => {
    const dashboard = useSelector((state: RootState) => state.dashboard, shallowEqual);
    const install = useSelector((state: RootState) => state.install, shallowEqual);

    if (!dashboard && !install) {
        return null;
    }

    const version = dashboard?.dnsVersion || install?.dnsVersion;

    

    return (
        <div className="version">
            <div className="version__text">
                {version && (
                    <>
                        <Trans>version</Trans>:&nbsp;
                        <span className="version__value" title={version}>
                            {version}
                        </span>
                    </>
                )}

                
            </div>
        </div>
    );
};

export default Version;
