import React from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { Controller, useFormContext } from 'react-hook-form';
import { ClientForm } from '../types';
import { BlockedService } from '../../../../Filters/Services/Form';
import { ServiceField } from '../../../../Filters/Services/ServiceField';

type Props = {
    services: BlockedService[];
};

export const BlockedServices = ({ services }: Props) => {
    const { t } = useTranslation();
    const { watch, setValue, control } = useFormContext<ClientForm>();

    const useGlobalServices = watch('use_global_blocked_services');


    // 使用“点号占位符”避免 RHF 把点解析为嵌套对象
    const DOT_TOKEN = '__DOT__';
    const encodeKey = (id: string) => id.replace(/\./g, DOT_TOKEN);
    const toFieldPath = (id: string) => `blocked_services.${encodeKey(id)}`;
    const handleToggleAllServices = (isSelected: boolean) => {
        services.forEach((service: BlockedService) => setValue(toFieldPath(service.id) as any, isSelected));
    };

    return (
        <div title={t('block_services')}>
            <div className="form__group">
                <Controller
                    name="use_global_blocked_services"
                    control={control}
                    render={({ field }) => (
                        <ServiceField
                            {...field}
                            data-testid="clients_use_global_blocked_services"
                            placeholder={t('blocked_services_global')}
                            className="service--global"
                        />
                    )}
                />

                <div className="row mb-4">
                    <div className="col-6">
                        <button
                            type="button"
                            data-testid="clients_block_all"
                            className="btn btn-secondary btn-block"
                            disabled={useGlobalServices}
                            onClick={() => handleToggleAllServices(true)}>
                            <Trans>block_all</Trans>
                        </button>
                    </div>

                    <div className="col-6">
                        <button
                            type="button"
                            data-testid="clients_unblock_all"
                            className="btn btn-secondary btn-block"
                            disabled={useGlobalServices}
                            onClick={() => handleToggleAllServices(false)}>
                            <Trans>unblock_all</Trans>
                        </button>
                    </div>
                </div>
                {services.length > 0 && (
                    <div className="services">
                        {services.map((service: BlockedService) => (
                            <Controller
                                key={service.id}
                                name={toFieldPath(service.id) as any}
                                control={control}
                                render={({ field }) => (
                                    <ServiceField
                                        {...field}
                                        data-testid={`clients_service_${service.id}`}
                                        placeholder={service.name}
                                        disabled={useGlobalServices}
                                        icon={service.icon_svg}
                                        iconBase64={service.icon_base64}
                                    />
                                )}
                            />
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
};
