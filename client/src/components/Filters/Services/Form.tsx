import React from 'react';

import { Trans } from 'react-i18next';

import { Controller, useForm } from 'react-hook-form';

import { ServiceField } from './ServiceField';

// 将服务 ID 转换为 React Hook Form 可安全使用的字段路径：
// 使用 bracket+引号 语法，避免点号被解析为嵌套对象。
const DOT_TOKEN = '__DOT__';
const encodeKey = (id: string) => id.replace(/\./g, DOT_TOKEN);
const toFieldPath = (id: string) => `blocked_services.${encodeKey(id)}`;

export type BlockedService = {
    id: string;
    name: string;
    icon_svg?: string;
    icon_base64?: string;
};

type FormValues = {
    blocked_services: Record<string, boolean>;
};

interface FormProps {
    initialValues: Record<string, boolean>;
    blockedServices: BlockedService[];
    onSubmit: (values: FormValues) => void;
    processing: boolean;
    processingSet: boolean;
}

export const Form = ({ initialValues, blockedServices, processing, processingSet, onSubmit }: FormProps) => {
    const {
        handleSubmit,
        control,
        setValue,
        reset,
        formState: { isSubmitting },
    } = useForm<FormValues>({
        mode: 'onBlur',
        defaultValues: {
            blocked_services: Object.fromEntries(
                Object.entries(initialValues?.blocked_services || {}).map(([k, v]) => [encodeKey(k), v]),
            ),
        },
    });

    // 当 initialValues 变化时重置表单
    React.useEffect(() => {
        reset({
            blocked_services: Object.fromEntries(
                Object.entries(initialValues?.blocked_services || {}).map(([k, v]) => [encodeKey(k), v]),
            ),
        });
    }, [initialValues, reset]);

    const handleToggleAllServices = async (isSelected: boolean) => {
        blockedServices.forEach((service: BlockedService) => setValue(toFieldPath(service.id) as any, isSelected));
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            <div className="form__group">
                <div className="row mb-4">
                    <div className="col-6">
                        <button
                            type="button"
                            data-testid="blocked_services_block_all"
                            className="btn btn-secondary btn-block"
                            disabled={processing || processingSet}
                            onClick={() => handleToggleAllServices(true)}>
                            <Trans>block_all</Trans>
                        </button>
                    </div>

                    <div className="col-6">
                        <button
                            type="button"
                            data-testid="blocked_services_unblock_all"
                            className="btn btn-secondary btn-block"
                            disabled={processing || processingSet}
                            onClick={() => handleToggleAllServices(false)}>
                            <Trans>unblock_all</Trans>
                        </button>
                    </div>
                </div>

                <div className="services">
                    {blockedServices.map((service: BlockedService) => (
                        <Controller
                            key={service.id}
                            name={toFieldPath(service.id) as any}
                            control={control}
                            render={({ field }) => (
                                <ServiceField
                                    {...field}
                                    data-testid={`blocked_services_${service.id}`}
                                    placeholder={service.name}
                                    disabled={processing || processingSet}
                                    icon={service.icon_svg}
                                    iconBase64={service.icon_base64}
                                />
                            )}
                        />
                    ))}
                </div>
            </div>

            <div className="btn-list">
                <button
                    type="submit"
                    data-testid="blocked_services_save"
                    className="btn btn-success btn-standard btn-large"
                    disabled={isSubmitting || processing || processingSet}>
                    <Trans>save_btn</Trans>
                </button>
            </div>
        </form>
    );
};
