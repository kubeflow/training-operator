import React from "react";
import PropTypes from "prop-types";
import TextField from "material-ui/TextField";
import SelectField from "material-ui/SelectField";
import MenuItem from "material-ui/MenuItem";
import { Card, CardText, CardActions } from "material-ui/Card";
import FlatButton from "material-ui/FlatButton";

const volumeKinds = {
  "Host Path": 0,
  "Azure File": 1
  // "NFS": 2,
  // "GlusterFS": 3
};

class Volume extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      volumeKind: 0,
      name: "",
      mountPath: "",
      hostPath: "",
      secretName: "",
      shareName: "",
      subPath: ""
    };
    this.handleInputChange = this.handleInputChange.bind(this);
    this.handleDelete = this.handleDelete.bind(this);
  }

  handleInputChange(event) {
    const target = event.target;
    const value = target.value;
    const name = target.name;
    this.setState({
      [name]: value
    });
    this.bubbleSpec({ ...this.state, [name]: value });
  }

  render() {
    return (
      <Card>
        <CardText style={this.styles.content}>
          <div style={this.styles.rowDirection}>
            <SelectField
              floatingLabelText="Kind"
              value={this.state.volumeKind}
              onChange={(o, v) => {
                this.setState({ volumeKind: v });
                this.bubbleSpec({ ...this.state, volumeKind: v });
              }}
              style={this.styles.element}
            >
              {Object.keys(volumeKinds).map((i, k) => (
                <MenuItem value={k} primaryText={i} key={i} />
              ))}
            </SelectField>
            <TextField
              floatingLabelText="Name"
              name="name"
              value={this.state.name}
              onChange={this.handleInputChange}
              style={this.styles.element}
            />
          </div>
          <div style={this.styles.rowDirection}>
            <TextField
              floatingLabelText="Mount Path"
              name="mountPath"
              value={this.state.mountPath}
              onChange={this.handleInputChange}
              style={this.styles.element}
            />
            <TextField
              floatingLabelText="Sub Path"
              name="subPath"
              value={this.state.subPath}
              onChange={this.handleInputChange}
              style={this.styles.element}
            />
          </div>
          {this.getFields()}
        </CardText>
        <CardActions>
          <FlatButton
            label="Delete"
            secondary={true}
            onClick={this.handleDelete}
          />
        </CardActions>
      </Card>
    );
  }

  styles = {
    content: {
      display: "flex",
      flexDirection: "column"
    },
    rowDirection: {
      flexDirection: "row"
    },
    element: {
      marginRight: "36px"
    }
  };

  getFields() {
    let child = {};
    switch (this.state.volumeKind) {
      case 0:
        child = (
          <TextField
            floatingLabelText="Host Path"
            name="hostPath"
            value={this.state.hostPath}
            onChange={this.handleInputChange}
            style={this.styles.element}
          />
        );
        break;
      case 1:
        child = (
          <div style={this.styles.rowDirection}>
            <TextField
              floatingLabelText="Secret Name"
              name="secretName"
              value={this.state.secretName}
              onChange={this.handleInputChange}
              style={this.styles.element}
            />
            <TextField
              floatingLabelText="Share Name"
              name="shareName"
              value={this.state.shareName}
              onChange={this.handleInputChange}
              style={this.styles.element}
            />
          </div>
        );
        break;
      default:
        console.error("error determining volume kind");
    }
    return child;
  }

  bubbleSpec(state) {
    let specificConfig = this.getSpecificVolumeSpec(state);
    const volumeSpec = Object.assign(
      {
        name: state.name,
        subPath: state.subPath
      },
      specificConfig
    );

    const volumeMountSpec = {
      name: state.name,
      mountPath: state.mountPath
    };
    this.props.setVolumeSpec(this.props.id, volumeSpec, volumeMountSpec);
  }

  getSpecificVolumeSpec(state) {
    switch (state.volumeKind) {
      case 0: //Host Path
        return {
          hostPath: {
            path: state.hostPath
          }
        };
      case 1: //Azure File
        return {
          azureFile: {
            secretName: state.secretName,
            shareName: state.shareName
          }
        };
      default:
        return {};
    }
  }

  handleDelete() {
    this.props.deleteVolume(this.props.id);
  }
}

Volume.propTypes = {
  id: PropTypes.string.isRequired,
  deleteVolume: PropTypes.func.isRequired,
  setVolumeSpec: PropTypes.func.isRequired
};

export default Volume;
