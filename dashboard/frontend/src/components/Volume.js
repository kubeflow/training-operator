import React from 'react';
import InfoEntry from './InfoEntry'
import TextField from 'material-ui/TextField';
import SelectField from 'material-ui/SelectField';
import MenuItem from 'material-ui/MenuItem';
import { Card, CardText } from 'material-ui/Card';
import { Tabs, Tab } from 'material-ui/Tabs';


const volumeKinds = {
    "Host Path": 0,
    "Azure File": 1,
    // "NFS": 2,
    // "GlusterFS": 3
}

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
    }

    handleInputChange(event) {
        const target = event.target;
        const value = target.value;
        const name = target.name;
        this.setState({
            [name]: value
        });
        this.bubbleSpec({...this.state, [name]: value});
    }

    render() {
        return (
            <Card>
                <CardText style={this.styles.content}>
                    <SelectField floatingLabelText="Kind" value={this.state.volumeKind} onChange={(o, v) => {
                        this.setState({ volumeKind: v });
                        this.bubbleSpec({ ...this.state, volumeKind: v });
                    }}>
                        {Object.keys(volumeKinds).map((i, k) => <MenuItem value={k} primaryText={i} key={i} />)}
                    </SelectField>
                    <TextField floatingLabelText="Name" name="name" value={this.state.name} onChange={this.handleInputChange} />
                    <TextField floatingLabelText="Mount Path" name="mountPath" value={this.state.mountPath} onChange={this.handleInputChange} />
                    <TextField floatingLabelText="Sub Path" name="subPath" value={this.state.subPath} onChange={this.handleInputChange} />
                    {this.getFields()}
                </CardText>
            </Card>
        );
    }

    styles = {
        content: {
            display: "flex",
            flexDirection: "column"
        }
    }

    getFields() {
        let children = [];
        switch (this.state.volumeKind) {
            case 0:
                children.push(<TextField floatingLabelText="Host Path" name="hostPath" key={0} value={this.state.hostPath} onChange={this.handleInputChange} />);
                break;
            case 1:
                children.push(<TextField floatingLabelText="Secret Name" name="secretName" key={0} value={this.state.secretName} onChange={this.handleInputChange} />);
                children.push(<TextField floatingLabelText="Share Name" name="shareName" key={1} value={this.state.shareName} onChange={this.handleInputChange} />);
                break;
        }
        return children;
    }

    bubbleSpec(state) {
        let specificConfig = this.getSpecificVolumeSpec(state)
        const volumeSpec = Object.assign({
            name: state.name,
            subPath: state.subPath
        }, specificConfig);

        const volumeMountSpec = {
            name: state.name,
            mountPath: state.mountPath
        }
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
}

export default Volume;