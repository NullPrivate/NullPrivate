import React, { Component, Fragment } from 'react';
import { Trans, withTranslation } from 'react-i18next';

import Table from './Table';

import Modal from './Modal';

import Card from '../../ui/Card';

import PageTitle from '../../ui/PageTitle';
import { MODAL_TYPE } from '../../../helpers/constants';
import { RewritesData } from '../../../initialState';

interface RewritesProps {
    t: (...args: unknown[]) => string;
    getRewritesList: () => (dispatch: any) => void;
    toggleRewritesModal: (...args: unknown[]) => unknown;
    addRewrite: (...args: unknown[]) => unknown;
    deleteRewrite: (...args: unknown[]) => unknown;
    updateRewrite: (...args: unknown[]) => unknown;
    rewrites: RewritesData;
}

interface RewritesState {
    ddnsDomain: string;
}

class Rewrites extends Component<RewritesProps, RewritesState> {
    constructor(props: RewritesProps) {
        super(props);
        this.state = {
            ddnsDomain: 'nas.home',
        };
    }

    componentDidMount() {
        this.props.getRewritesList();
    }

    handleDelete = (values: any) => {
        // eslint-disable-next-line no-alert
        if (window.confirm(this.props.t('rewrite_confirm_delete', { key: values.domain }))) {
            this.props.deleteRewrite(values);
        }
    };

    handleSubmit = (values: any) => {
        const { modalType, currentRewrite } = this.props.rewrites;

        if (modalType === MODAL_TYPE.EDIT_REWRITE && currentRewrite) {
            this.props.updateRewrite({
                target: currentRewrite,
                update: values,
            });
        } else {
            this.props.addRewrite(values);
        }
    };

    handleDDNSDomainChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        this.setState({ ddnsDomain: event.target.value });
    };

    downloadDDNSScript = (os: string) => {
        const { ddnsDomain } = this.state;
        const domainParam = ddnsDomain ? `?domain=${encodeURIComponent(ddnsDomain)}` : '';
        const endpoint = `/control/ddns/script/${os}${domainParam}`;
        window.location.href = endpoint;
    };

    render() {
        const {
            t,

            rewrites,

            toggleRewritesModal,
        } = this.props;

        const {
            list,
            isModalOpen,
            processing,
            processingAdd,
            processingDelete,
            processingUpdate,
            modalType,
            currentRewrite,
        } = rewrites;

        const { ddnsDomain } = this.state;

        return (
            <Fragment>
                <PageTitle title={t('dns_rewrites')} subtitle={t('rewrite_desc')} />

                <Card id="rewrites" bodyType="card-body box-body--settings">
                    <Fragment>
                        <Table
                            list={list}
                            processing={processing}
                            processingAdd={processingAdd}
                            processingDelete={processingDelete}
                            processingUpdate={processingUpdate}
                            handleDelete={this.handleDelete}
                            toggleRewritesModal={toggleRewritesModal}
                        />

                        <button
                            type="button"
                            className="btn btn-success btn-standard mt-3"
                            onClick={() => toggleRewritesModal({ type: MODAL_TYPE.ADD_REWRITE })}
                            disabled={processingAdd}>
                            <Trans>rewrite_add</Trans>
                        </button>

                        {/* DDNS 脚本下载区域 */}
                        <div className="mt-4">
                            <h4><Trans>ddns_scripts</Trans></h4>
                            <p className="text-muted mt-2"><Trans>ddns_scripts_desc</Trans></p>
                            <div className="form-group">
                                <label htmlFor="ddnsDomain" className="form-label">
                                    <Trans>ddns_domain</Trans>
                                </label>
                                <input
                                    type="text"
                                    id="ddnsDomain"
                                    className="form-control"
                                    value={ddnsDomain}
                                    onChange={this.handleDDNSDomainChange}
                                    placeholder={t('ddns_domain_placeholder')}
                                />
                                <small className="form-text text-muted">
                                    <Trans>ddns_domain_help</Trans>
                                </small>
                            </div>
                            <div className="btn-group mt-3">
                                <button
                                    type="button"
                                    className="btn btn-outline-primary"
                                    onClick={() => this.downloadDDNSScript('windows')}
                                    disabled={!ddnsDomain}>
                                    <Trans>ddns_script_windows</Trans>
                                </button>
                                <button
                                    type="button"
                                    className="btn btn-outline-primary"
                                    onClick={() => this.downloadDDNSScript('linux')}
                                    disabled={!ddnsDomain}>
                                    <Trans>ddns_script_linux</Trans>
                                </button>
                                <button
                                    type="button"
                                    className="btn btn-outline-primary"
                                    onClick={() => this.downloadDDNSScript('macos')}
                                    disabled={!ddnsDomain}>
                                    <Trans>ddns_script_macos</Trans>
                                </button>
                            </div>
                            
                            {/* Script execution instructions */}
                            <div className="mt-4">
                                <h5><Trans>ddns_script_instructions</Trans></h5>
                                <p className="text-muted"><Trans>ddns_script_instructions_desc</Trans></p>

                                <h6 className="mt-3"><Trans>ddns_script_windows</Trans></h6>
                                <p className="text-muted"><Trans>ddns_script_windows_note</Trans></p>
                                <pre><code>
                                    {'Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope Process\n' 
                                        + '.\\ddns-script.ps1'}
                                </code></pre>

                                <h6 className="mt-3"><Trans>ddns_script_linux</Trans> / <Trans>ddns_script_macos</Trans></h6>
                                <p className="text-muted"><Trans>ddns_script_unix_note</Trans></p>
                                <pre><code>
                                    {'chmod +x ddns-script.sh\n' 
                                        + './ddns-script.sh'}
                                </code></pre>
                            </div>
                            
                        </div>

                        <Modal
                            isModalOpen={isModalOpen}
                            modalType={modalType}
                            toggleRewritesModal={toggleRewritesModal}
                            handleSubmit={this.handleSubmit}
                            processingAdd={processingAdd}
                            processingDelete={processingDelete}
                            currentRewrite={currentRewrite}
                        />
                    </Fragment>
                </Card>
            </Fragment>
        );
    }
}

export default withTranslation()(Rewrites);
