import React from 'react';
import { Card, CardHeader, CardText } from 'material-ui/Card';
import FlatButton from 'material-ui/FlatButton';
import ContentAdd from 'material-ui/svg-icons/content/add';

import Volume from './Volume';

class VolumeCreator extends React.Component {

  constructor(props) {
    super(props);
    this.state = {
      volumeSpecs: {},
      volumeCount: 0
    };

    this.setVolumeSpec = this.setVolumeSpec.bind(this);
  }

  render() {
    return (
      <Card style={this.styles.card} expanded={true}>
        <CardHeader
          title="Volumes"
          actAsExpander={true}
          showExpandableButton={true} >
          <FlatButton>
            <ContentAdd onClick={_ => this.addVolume()} />
          </FlatButton>
        </CardHeader>
        <CardText expandable={true}>
          {Object.keys(this.state.volumeSpecs).map(k => <Volume key={k} id={k} setVolumeSpec={this.setVolumeSpec} />)}
        </CardText>
      </Card>
    );
  }

  setVolumeSpec(id, volumeSpec, volumeMountSpec) {
    let spec = {
      volume: volumeSpec,
      volumeMount: volumeMountSpec
    }
    let volumeSpecs = { ...this.state.volumeSpecs, [id]: spec };
    this.setState({ volumeSpecs: volumeSpecs });
    this.bubbleSpecs(volumeSpecs)
  }

  bubbleSpecs(volumeSpecs) {
    let specs = { volumes: [], volumeMounts: [] };
    Object.keys(this.state.volumeSpecs).map( k => {
      const v = this.state.volumeSpecs[k]
      specs.volumes.push(v.volume);
      specs.volumeMounts.push(v.volumeMount)
    });
    this.props.setVolumesSpec(specs)
  }

  addVolume() {
    const id = this.state.volumeCount;
    this.setState({ volumeCount: id + 1, volumeSpecs: { ...this.state.volumeSpecs, [id]: {} } })
  }

  styles = {
    card: {
      boxShadow: ""
    }
  }
}

export default VolumeCreator;