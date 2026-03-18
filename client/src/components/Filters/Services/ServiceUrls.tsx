import React, { useEffect, useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { useTranslation } from 'react-i18next';
import cn from 'classnames';
import { getServiceUrls, updateServiceUrls, reloadBlockedServices } from '../../../actions/services';
import Card from '../../ui/Card';

const ServiceUrls = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const {
        serviceUrls,
        processingGetUrls,
        processingSetUrls,
        processingReload,
    } = useSelector((state: any) => state.services);
    const [showInputs, setShowInputs] = useState(false);
    const [urls, setUrls] = useState<string[]>([]);

    useEffect(() => {
        dispatch(getServiceUrls());
    }, [dispatch]);

    useEffect(() => {
        if (serviceUrls) {
            setUrls(serviceUrls);
        }
    }, [serviceUrls]);

    const handleAddUrl = () => {
        setUrls([...urls, '']);
    };

    const handleRemoveUrl = (index: number) => {
        const newUrls = urls.filter((_, i) => i !== index);
        setUrls(newUrls);
    };

    const handleUrlChange = (index: number, value: string) => {
        const newUrls = [...urls];
        newUrls[index] = value;
        setUrls(newUrls);
    };

    const handleReloadServices = () => {
        dispatch(reloadBlockedServices());
    };

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        // 过滤掉空字符串
        const filteredUrls = urls.filter(url => url.trim() !== '');
        dispatch(updateServiceUrls(filteredUrls));
    };

    if (processingGetUrls) {
        return <div>{t('loading_service_urls')}</div>;
    }

    return (
        <Card
            bodyType="card-body box-body--settings"
            title={t('service_urls')}
            subtitle={t('service_urls_desc')}
            id="service-urls"
        >
            <div className="mb-4">
                <div className="d-flex flex-wrap align-items-center mb-2">
                    <button
                        type="button"
                        className={cn('btn mr-2 mb-2', showInputs ? 'btn-outline-primary' : 'btn-primary')}
                        onClick={() => setShowInputs(!showInputs)}
                    >
                        {showInputs ? t('hide_urls') : t('show_urls')}
                    </button>
                    <a
                        href="https://list.adguardprivate.com"
                        target="_blank"
                        rel="noopener noreferrer"
                        className="btn btn-outline-primary mb-2"
                    >
                        {t('online_generate_anti_addiction_list', {
                            defaultValue: 'Generate anti-addiction list online',
                        })}
                    </a>
                </div>
                {showInputs && (
                    <form onSubmit={handleSubmit}>
                        <div className="row">
                            <div className="col-12">
                                {urls.map((url, index) => (
                                    <div
                                        key={index}
                                        className="d-flex align-items-center mb-2"
                                    >
                                        <input
                                            type="text"
                                            className="form-control w-100"
                                            placeholder={t('service_url_placeholder')}
                                            value={url}
                                            onChange={(e) => handleUrlChange(index, e.target.value)}
                                        />
                                        <button
                                            type="button"
                                            className="btn btn-outline-secondary ml-2"
                                            onClick={() => handleRemoveUrl(index)}
                                        >
                                            {t('delete')}
                                        </button>
                                    </div>
                                ))}
                                <div className="d-flex justify-content-between mt-3">
                                    <button
                                        type="button"
                                        className="btn btn-primary mr-2"
                                        onClick={handleAddUrl}
                                    >
                                        {t('add_url')}
                                    </button>
                                    <div className="d-flex">
                                        <button
                                            type="button"
                                            className={cn('btn btn-primary mr-2', { 'btn--loading': processingReload })}
                                            onClick={handleReloadServices}
                                            disabled={processingReload}
                                        >
                                            {t('reload_services')}
                                        </button>
                                        <button
                                            type="submit"
                                            className={cn('btn btn-success', { 'btn--loading': processingSetUrls })}
                                            disabled={processingSetUrls}
                                        >
                                            {t('save')}
                                        </button>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </form>
                )}
            </div>
        </Card>
    );
};

export default ServiceUrls;
