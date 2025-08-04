import React from 'react';
import { useTranslation } from 'react-i18next';
import { Controller, useFormContext } from 'react-hook-form';
import { useSelector } from 'react-redux';
import i18next from 'i18next';
import { captitalizeWords } from '../../../../../helpers/helpers';
import { ClientForm } from '../types';
import { Checkbox } from '../../../../ui/Controls/Checkbox';
import { RootState } from '../../../../../initialState';

type ProtectionSettings = 'use_global_settings' | 'filtering_enabled' | 'safebrowsing_enabled' | 'parental_enabled';

const settingsCheckboxes: {
    name: ProtectionSettings;
    placeholder: string;
    requiredServiceTypes?: string[];
}[] = [
    {
        name: 'use_global_settings',
        placeholder: i18next.t('client_global_settings'),
        requiredServiceTypes: ['family', 'enterprise'],
    },
    {
        name: 'filtering_enabled',
        placeholder: i18next.t('block_domain_use_filters_and_hosts'),
        requiredServiceTypes: ['family', 'enterprise'],
    },
    {
        name: 'safebrowsing_enabled',
        placeholder: i18next.t('use_adguard_browsing_sec'),
        requiredServiceTypes: ['family', 'enterprise'],
    },
    {
        name: 'parental_enabled',
        placeholder: i18next.t('use_adguard_parental'),
        requiredServiceTypes: ['family', 'enterprise'],
    },
];

type LogsStatsSettings = 'ignore_querylog' | 'ignore_statistics';

const logAndStatsCheckboxes: { name: LogsStatsSettings; placeholder: string }[] = [
    {
        name: 'ignore_querylog',
        placeholder: i18next.t('ignore_query_log'),
    },
    {
        name: 'ignore_statistics',
        placeholder: i18next.t('ignore_statistics'),
    },
];

type Props = {
    safeSearchServices: Record<string, boolean>;
};

export const MainSettings = ({ safeSearchServices }: Props) => {
    const { t } = useTranslation();
    const { watch, control } = useFormContext<ClientForm>();
    const serviceType = useSelector((state: RootState) => state.service_type);

    const useGlobalSettings = watch('use_global_settings');

    // Filter settings based on service type
    const filteredSettingsCheckboxes = settingsCheckboxes.filter(setting => {
        if (!setting.requiredServiceTypes || setting.requiredServiceTypes.length === 0) {
            return true;
        }
        return setting.requiredServiceTypes.includes(serviceType);
    });

    // Check if safe search should be shown for current service type
    const shouldShowSafeSearch = serviceType === 'family' || serviceType === 'enterprise';

    return (
        <div title={t('main_settings')}>
            <div className="form__label--bot form__label--bold">{t('protection_section_label')}</div>
            {filteredSettingsCheckboxes.map((setting) => (
                <div className="form__group" key={setting.name}>
                    <Controller
                        name={setting.name}
                        control={control}
                        render={({ field }) => (
                            <Checkbox
                                {...field}
                                data-testid={`clients_${setting.name}`}
                                title={setting.placeholder}
                                disabled={setting.name !== 'use_global_settings' ? useGlobalSettings : false}
                            />
                        )}
                    />
                </div>
            ))}

            {shouldShowSafeSearch && (
                <>
            <div className="form__group">
                <Controller
                    name="safe_search.enabled"
                    control={control}
                    render={({ field }) => (
                        <Checkbox
                            data-testid="clients_safe_search"
                            {...field}
                            title={t('enforce_safe_search')}
                            disabled={useGlobalSettings}
                        />
                    )}
                />
            </div>

            <div className="form__group--inner">
                {Object.keys(safeSearchServices).map((searchKey) => (
                    <div key={searchKey}>
                        <Controller
                            name={`safe_search.${searchKey}`}
                            control={control}
                            render={({ field }) => (
                                <Checkbox
                                    {...field}
                                    data-testid={`clients_safe_search_${searchKey}`}
                                    title={captitalizeWords(searchKey)}
                                    disabled={useGlobalSettings}
                                />
                            )}
                        />
                    </div>
                ))}
            </div>
                </>
            )}

            <div className="form__label--bold form__label--top form__label--bot">
                {t('log_and_stats_section_label')}
            </div>
            {logAndStatsCheckboxes.map((setting) => (
                <div className="form__group" key={setting.name}>
                    <Controller
                        name={setting.name}
                        control={control}
                        render={({ field }) => (
                            <Checkbox {...field} data-testid={`clients_${setting.name}`} title={setting.placeholder} />
                        )}
                    />
                </div>
            ))}
        </div>
    );
};
