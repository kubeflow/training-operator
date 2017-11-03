import React, { Component } from 'react';
import { Card, CardTitle, CardText, CardActions } from 'material-ui/Card';
import TextField from 'material-ui/TextField';
import Divider from 'material-ui/Divider';
import SelectField from 'material-ui/SelectField';
import MenuItem from 'material-ui/MenuItem';
import Toggle from 'material-ui/Toggle';
import RaisedButton from 'material-ui/RaisedButton';
import {
  withRouter
} from 'react-router-dom'

class CreateJob extends Component {

  constructor(props) {
    super(props)
    this.state = {
      name: "",
      namespace: "default",
      masterImage: "",
      masterGpuCount: 0,
      workerImage: "",
      workerReplicas: 1,
      workerGpuCount: 0,
      psUseDefaultImage: true,
      psReplicas: 0,
      psImage: "",
      tbIsPresent: true,
      tbServiceType: 0
    };

    this.handleInputChange = this.handleInputChange.bind(this);
    this.cancel = this.cancel.bind(this);
    this.deploy = this.deploy.bind(this);
  }

  handleInputChange(event) {
    const target = event.target;
    const value = target.type === 'checkbox' ? target.checked : target.value;
    const name = target.name;
    this.setState({
      [name]: value
    });
  }

  styles = {
    divider: {
      marginTop: "20px",
      marginBottom: "5px"
    },
    header: {
      fontWeight: "bold",
      marginTop: "10px"
    },
    toggle: {
      width: "260px",
      marginTop: "10px"

    },
    root: {
      display: "flex",
      flexDirection: "column"
    }
  }
  render() {
    return (
      <Card>
        <CardTitle title="Start a New Training" />
        <CardText style={this.styles.root}>
          <TextField floatingLabelText="Training name" name="name" onChange={this.handleInputChange} />
          <TextField floatingLabelText="Namespace" name="namespace" value={this.state.namespace} onChange={this.handleInputChange} />


          {/* MASTER */}
          <Divider style={this.styles.divider} />
          <p style={this.styles.header} >Master</p>
          <TextField floatingLabelText="Container image" name="masterImage" value={this.state.masterImage} onChange={this.handleInputChange} />
          <TextField floatingLabelText="GPU(s) per replica" type="number" min="0" name="masterGpuCount" value={this.state.masterGpuCount} onChange={this.handleInputChange} />

          {/* WORKER */}
          <Divider style={this.styles.divider} />
          <p style={this.styles.header}>Worker(s)</p>
          <TextField floatingLabelText="Container image" name="workerImage" value={this.state.workerImage} onChange={this.handleInputChange} />
          <TextField floatingLabelText="Replicas" type="number" min="0" name="workerReplicas" value={this.state.workerReplicas} onChange={this.handleInputChange} />
          <TextField floatingLabelText="GPU(s) per replica" type="number" min="0" name="workerGpuCount" value={this.state.workerGpuCount} onChange={this.handleInputChange} />

          {/* PARAMETER SERVER */}
          <Divider style={this.styles.divider} />
          <p style={this.styles.header}>Parameter Server(s)</p>
          <TextField floatingLabelText="Replicas" name="psReplicas" type="number" min="0" value={this.state.psReplicas} onChange={this.handleInputChange} />
          <Toggle label="Use default image" defaultToggled={true} name="psUseDefaultImage" onToggle={this.handleInputChange} style={this.styles.toggle} />
          {!this.state.psUseDefaultImage &&
            <TextField floatingLabelText="Container image" name="psImage" value={this.state.psImage} onChange={this.handleInputChange} />
          }

          {/* TENSORBOARD */}
          <Divider style={this.styles.divider} />
          <Toggle label="TensorBoard" defaultToggled={true} name="tbIsPresent" onToggle={this.handleInputChange} style={this.styles.toggle} />
          {this.state.tbIsPresent &&
            <SelectField floatingLabelText="Service" value={this.state.tbServiceType} onChange={(o, v) => this.setState({ tbServiceType: v })}>
              <MenuItem value={0} primaryText="Internal" />
              <MenuItem value={1} primaryText="External" />
            </SelectField>
          }
        </CardText>
        <CardActions>
          <RaisedButton label="Deploy" primary={true} onClick={this.deploy} />
          <RaisedButton label="Cancel" onClick={this.cancel} />
        </CardActions>
      </Card >
    );
  }

  deploy() {

    let rs = [
      this.newReplicaSpec("MASTER", 1, this.state.masterImage)
    ]
    if (this.state.workerReplicas > 0) {
      rs.push(this.newReplicaSpec("WORKER", this.state.workerReplicas, this.state.workerImage));
    }
    if (this.state.psReplicas > 0) {
      rs.push(this.newReplicaSpec("PS", this.state.psReplicas, this.state.psImage));
    }

    let spec = {
      metadata: {
        name: this.state.name,
        namespace: this.state.namespace
      },
      spec: {
        replicaSpecs: rs
      }
    }

    let myHeaders = new Headers();
    myHeaders.append("Content-Type", "application/json");
    const options = {
      method: "POST",
      headers: myHeaders,
      body: JSON.stringify(spec)
    };

    console.log(spec)
    fetch(`http://localhost:8080/api/tfjob`, options)
      .then(r => r.json())
      .then(console.log);

  }

  cancel() {
    this.props.history.goBack();
  }

  newReplicaSpec(tfReplicaType, replicas, image) {
    return {
      replicas: parseInt(replicas),
      tfReplicaType,
      template: {
        spec: {
          containers: [{
            image,
            name: "tensorflow"
          }],
          restartPolicy: "OnFailure"
        }
      }

    }
  }
}

export default withRouter(CreateJob);
