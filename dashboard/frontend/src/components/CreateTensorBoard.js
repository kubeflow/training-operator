import React, { Component } from 'react';
import TextField from 'material-ui/TextField';
import SelectField from 'material-ui/SelectField';
import MenuItem from 'material-ui/MenuItem';
import Volume from './Volume'

class CreateTensorBoard extends Component {

    constructor(props) {
        super(props)
        this.state = {
            serviceType: 0,
            logDir: "/tmp/tensorflow"
        };
        this.handleInputChange = this.handleInputChange.bind(this);
    }

    bubbleSpec(state) {
        // Send data to parent component
        this.props.setTensorBoardSpec(this.buildTensorBoardSpec(state))
    }

    handleInputChange(event) {
        const target = event.target;
        const value = target.type === 'checkbox' ? target.checked : target.value;
        const name = target.name;
        this.setState({
            [name]: value
        });
        this.bubbleSpec({...this.state, [name]: value});
    }

    styles = {
        root: {
            display: "flex",
            flexDirection: "column"
        },
        field: {
            width: "80%"
        }
    }
    render() {
        // this.bubbleSpec();
        return (
            <div style={this.styles.root}>
                <SelectField style={this.styles.field}  floatingLabelText="Service" value={this.state.serviceType} onChange={(o, v) => {
                    this.setState({ serviceType: v });
                    this.bubbleSpec({...this.state, serviceType: v});
                    }}>
                    <MenuItem value={0} primaryText="Internal" />
                    <MenuItem value={1} primaryText="External" />
                </SelectField>
                <TextField style={this.styles.field}  floatingLabelText="Log dir" name="logDir" value={this.state.logDir} onChange={this.handleInputChange} />
                {/* <Volume /> */}
                {/* <VolumeMount /> */}
            </div >
        );
    }

    buildTensorBoardSpec(state) {
        // "tensorboard": {
        //   "logDir": "/tmp/tensorflow",
        //   "volumes": [
        //    {
        //     "name": "azurefile",
        //     "azureFile": {
        //      "secretName": "azure-secret",
        //      "shareName": "data"
        //     }
        //    }
        //   ],
        //   "volumeMounts": [
        //    {
        //     "name": "azurefile",
        //     "mountPath": "/tmp/tensorflow"
        //    }
        //   ],
        //   "serviceType": "LoadBalancer"
        //  }
        let st = state.serviceType == 0 ? "ClusterIP" : "LoadBalancer";
        return { "logDir": state.logDir, "serviceType": st };
    }
}

export default CreateTensorBoard;
