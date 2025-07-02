import React from 'react';
import { Controller, useForm } from 'react-hook-form';
import { Trans, useTranslation } from 'react-i18next';
import { useSelector } from 'react-redux';

import { removeEmptyLines } from '../../../../helpers/helpers';
import { RootState } from '../../../../initialState';
import { Textarea } from '../../../ui/Controls/Textarea';

type FormData = {
    upstream_alternate_rulesets: string;
    upstream_alternate_dns: string;
};

type FormProps = {
    initialValues?: Partial<FormData>;
    onSubmit: (data: FormData) => void;
};

const Form = ({ initialValues, onSubmit }: FormProps) => {
    const { t } = useTranslation();

    const {
        control,
        handleSubmit,
        formState: { isSubmitting, isDirty },
    } = useForm<FormData>({
        mode: 'onBlur',
        defaultValues: {
            upstream_alternate_rulesets: initialValues?.upstream_alternate_rulesets || '',
            upstream_alternate_dns: initialValues?.upstream_alternate_dns || '',
        },
    });

    const processingSetConfig = useSelector((state: RootState) => state.dnsConfig.processingSetConfig);
    const processingTestUpstream = useSelector((state: RootState) => state.settings.processingTestUpstream);

    return (
        <form onSubmit={handleSubmit(onSubmit)} className="form--alt-upstream">
            <div className="row">
                {/* Alternate upstream section */}
                <div className="col-12 mb-2">
                    <label className="form__label form__label--with-desc" htmlFor="upstream_alternate_rulesets">
                        <Trans>upstream_alternate_rulesets_title</Trans>
                    </label>
                    <div className="form__desc form__desc--top">
                        <Trans>upstream_alternate_rulesets_desc</Trans>
                    </div>
                    <Controller
                        name="upstream_alternate_rulesets"
                        control={control}
                        render={({ field }) => (
                            <Textarea
                                {...field}
                                id="upstream_alternate_rulesets"
                                className="form-control form-control--textarea form-control--textarea-small font-monospace"
                                placeholder={t('upstream_alternate_rulesets_placeholder')}
                                disabled={processingSetConfig}
                                onBlur={(e) => {
                                    const value = removeEmptyLines(e.target.value);
                                    field.onChange(value);
                                }}
                            />
                        )}
                    />
                </div>

                <div className="col-12">
                    <label className="form__label form__label--with-desc" htmlFor="upstream_alternate_dns">
                        <Trans>upstream_alternate_dns_title</Trans>
                    </label>
                    <div className="form__desc form__desc--top">
                        <Trans>upstream_alternate_dns_desc</Trans>
                    </div>
                    <Controller
                        name="upstream_alternate_dns"
                        control={control}
                        render={({ field }) => (
                            <Textarea
                                {...field}
                                id="upstream_alternate_dns"
                                className="form-control form-control--textarea form-control--textarea-small font-monospace"
                                placeholder={t('upstream_alternate_dns_placeholder')}
                                disabled={processingSetConfig || processingTestUpstream}
                                onBlur={(e) => {
                                    const value = removeEmptyLines(e.target.value);
                                    field.onChange(value);
                                }}
                            />
                        )}
                    />
                </div>
            </div>

            <div className="card-actions">
                <div className="btn-list">
                    <button
                        type="submit"
                        data-testid="alt_upstream_save"
                        className="btn btn-success btn-standard"
                        disabled={isSubmitting || !isDirty || processingSetConfig || processingTestUpstream}>
                        {t('apply_btn')}
                    </button>
                </div>
            </div>
        </form>
    );
};

export default Form;