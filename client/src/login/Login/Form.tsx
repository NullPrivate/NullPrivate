import React, { useEffect } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { Input } from '../../components/ui/Controls/Input';
import { validateRequiredValue } from '../../helpers/validators';

export type LoginFormValues = {
    username: string;
    password: string;
};

type LoginFormProps = {
    onSubmit: (data: LoginFormValues) => void;
    processing: boolean;
};

// Function to extract subdomain
const extractSubdomain = (): string => {
    const {hostname} = window.location;
    const parts = hostname.split('.');

    // Check if the format is xxx.adguardprivate.com
    if (parts.length >= 3 && parts[parts.length - 2] === 'adguardprivate' && parts[parts.length - 1] === 'com') {
        return parts[0]; // Return the first part as subdomain
    }

    return ''; // Return empty string if format does not match
};

const Form = ({ onSubmit, processing }: LoginFormProps) => {
    const { t } = useTranslation();
    const {
        handleSubmit,
        control,
        setValue,
        formState: { isValid },
    } = useForm<LoginFormValues>({
        mode: 'onChange',
        defaultValues: {
            username: '',
            password: '',
        },
    });

    // Auto-fill username
    useEffect(() => {
        const subdomain = extractSubdomain();
        if (subdomain) {
            setValue('username', subdomain, { shouldValidate: true });
        }
    }, [setValue]);

    return (
        <form onSubmit={handleSubmit(onSubmit)} className="card">
            <div className="card-body p-6">
                <div className="form__group form__group--settings">
                    <Controller
                        name="username"
                        control={control}
                        rules={{ validate: validateRequiredValue }}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                data-testid="username"
                                type="text"
                                label={t('username_label')}
                                placeholder={t('username_placeholder')}
                                error={fieldState.error?.message}
                                autoComplete="username"
                                autoCapitalize="none"
                                disabled={processing}
                            />
                        )}
                    />
                </div>

                <div className="form__group form__group--settings">
                    <Controller
                        name="password"
                        control={control}
                        rules={{ validate: validateRequiredValue }}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                data-testid="password"
                                type="password"
                                label={t('password_label')}
                                placeholder={t('password_placeholder')}
                                error={fieldState.error?.message}
                                autoComplete="current-password"
                                disabled={processing}
                            />
                        )}
                    />
                </div>

                <div className="form-footer">
                    <button
                        data-testid="sign_in"
                        type="submit"
                        className="btn btn-success btn-block"
                        disabled={processing || !isValid}>
                        {t('sign_in')}
                    </button>
                </div>
            </div>
        </form>
    );
};

export default Form;
