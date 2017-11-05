import React, { Component } from 'react';
import {
    withRouter
} from 'react-router-dom'

import { default as BaseAppBar } from 'material-ui/AppBar';
import FlatButton from 'material-ui/FlatButton';
import ActionDelete from 'material-ui/svg-icons/action/delete';
import Dialog from 'material-ui/Dialog';

import { deleteTfJob } from '../services';

class AppBar extends Component {

    constructor(props) {
        super(props);
        this.state = {
            isModalVisible: false
        }
    }


    render() {
        const actions = [
            <FlatButton
                label="CANCEL"
                primary={true}
                onClick={_ => this.setState({ isModalVisible: false })}
            />,
            <FlatButton
                label="DELETE"
                primary={true}
                onClick={_ => {
                    this.setState({ isModalVisible: false });
                    this.deleteJob();
                }}
            />
        ];

        let rightMenu = null;
        if (this.props.location.pathname != "/new") {
            rightMenu = (
                <div style={this.styles.rightMenu}>
                    <FlatButton default={true} label="DELETE" icon={<ActionDelete />} style={this.styles.rightMenuButton} onClick={_ => this.setState({ isModalVisible: true })} />
                </div>
            );
        }
        return (
            <div>
                <BaseAppBar
                    title={this.getTitle()}
                    iconElementRight={rightMenu}
                    showMenuIconButton={false}
                />
                <Dialog
                    title="Delete a TfJob"
                    actions={actions}
                    modal={true}
                    open={this.state.isModalVisible}>
                    Are you sure you want to delete TfJob "test" in namespace "test"?
                </Dialog>
            </div>
        );
    }

    getTitle() {
        const path = this.props.location.pathname;
        switch (path) {
            case "/new":
                return "New Training";
            default:
                return path.replace('/', '')
        }
        return this.props.location.pathname
    }

    deleteJob() {
        let path = this.props.location.pathname.replace('/', '');
        path = path.split('/');
        const ns = path[0];
        const name = path[1];

        let myHeaders = new Headers();
        myHeaders.append("Content-Type", "application/json");
        const options = {
            method: "DELETE",
            headers: myHeaders
            // body: JSON.stringify(spec)
        };

        deleteTfJob(ns, name)
            .catch(console.log);
    }


    styles = {
        rightMenu: {
            marginTop: "6px",
        },
        rightMenuButton: {
            color: "white",
            fontWeight: "bold !important"
        }
    }

}

export default withRouter(AppBar);
