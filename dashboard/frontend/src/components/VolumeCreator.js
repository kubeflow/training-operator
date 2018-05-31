import React from "react";
import PropTypes from "prop-types";
import FlatButton from "material-ui/FlatButton";
import ContentAdd from "material-ui/svg-icons/content/add";
import omit from "lodash/omit";

import Volume from "./Volume";

class VolumeCreator extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      volumeSpecs: {},
      volumeCount: 0
    };

    this.setVolumeSpec = this.setVolumeSpec.bind(this);
    this.deleteVolume = this.deleteVolume.bind(this);
  }


  styles = {
    root: {
      marginTop: "20px"
    }
  };
  render() {
    return (
      <div style={this.styles.root}>
        <h4>Volumes</h4>
        <FlatButton
          label="Add a volume"
          primary={true}
          icon={<ContentAdd />}
          onClick={() => this.addVolume()}
        />
        {Object.keys(this.state.volumeSpecs).map(k => (
          <Volume
            key={k}
            id={k}
            setVolumeSpec={this.setVolumeSpec}
            deleteVolume={this.deleteVolume}
          />
        ))}
      </div>
    );
  }

  setVolumeSpec(id, volumeSpec, volumeMountSpec) {
    let spec = {
      volume: volumeSpec,
      volumeMount: volumeMountSpec
    };
    let volumeSpecs = { ...this.state.volumeSpecs, [id]: spec };
    this.setState({ volumeSpecs: volumeSpecs });
    this.bubbleSpecs(volumeSpecs);
  }

  deleteVolume(id) {
    this.setState({ volumeSpecs: omit(this.state.volumeSpecs, [id]) });
  }

  bubbleSpecs(volumeSpecs) {
    let specs = { volumes: [], volumeMounts: [] };
    Object.keys(volumeSpecs).forEach(k => {
      const v = volumeSpecs[k];
      specs.volumes.push(v.volume);
      specs.volumeMounts.push(v.volumeMount);
    });
    this.props.setVolumesSpec(specs);
  }

  addVolume() {
    const id = this.state.volumeCount;
    this.setState({
      volumeCount: id + 1,
      volumeSpecs: { ...this.state.volumeSpecs, [id]: {} }
    });
  }
}

VolumeCreator.propTypes = {
  setVolumesSpec: PropTypes.func.isRequired
};

export default VolumeCreator;
