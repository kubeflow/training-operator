import React, { Component } from "react";
import PropTypes from "prop-types";
import { withRouter } from "react-router-dom";

import { default as BaseAppBar } from "material-ui/AppBar";
import FlatButton from "material-ui/FlatButton";
import ActionDelete from "material-ui/svg-icons/action/delete";
import Dialog from "material-ui/Dialog";

import { deleteTFJob } from "../services";

class AppBar extends Component {
  constructor(props) {
    super(props);
    this.state = {
      isModalVisible: false
    };
  }

  render() {
    const actions = [
      <FlatButton
        key={"cancel"}
        label="CANCEL"
        primary={true}
        onClick={() => this.setState({ isModalVisible: false })}
      />,
      <FlatButton
        key={"delete"}
        label="DELETE"
        primary={true}
        onClick={() => {
          this.setState({ isModalVisible: false });
          this.deleteJob();
        }}
      />
    ];

    let rightMenu = null;
    if (this.props.location.pathname !== "/new") {
      rightMenu = (
        <div style={this.styles.rightMenu}>
          <FlatButton
            default={true}
            label="DELETE"
            icon={<ActionDelete />}
            style={this.styles.rightMenuButton}
            onClick={() => this.setState({ isModalVisible: true })}
          />
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
          title="Delete a TFJob"
          actions={actions}
          modal={true}
          open={this.state.isModalVisible}
        >
          {'Are you sure you want to delete TFJob "test" in namespace "test"?'}
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
        return path.replace("/", "");
    }
  }

  deleteJob() {
    let path = this.props.location.pathname.replace("/", "");
    path = path.split("/");
    const ns = path[0];
    const name = path[1];

    deleteTFJob(ns, name).catch(console.error);
  }

  styles = {
    rightMenu: {
      marginTop: "6px"
    },
    rightMenuButton: {
      color: "white",
      fontWeight: "bold !important"
    }
  };
}

AppBar.propTypes = {
  location: PropTypes.shape({
    pathname: PropTypes.string.isRequired
  }).isRequired
};

export default withRouter(AppBar);
