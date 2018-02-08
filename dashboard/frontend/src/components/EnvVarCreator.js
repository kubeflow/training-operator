import React from "react";
import PropTypes from "prop-types";
import FlatButton from "material-ui/FlatButton";
import ContentAdd from "material-ui/svg-icons/content/add";
import omit from "lodash/omit";

import EnvVar from "./EnvVar";

class EnvVarCreator extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      envVars: {},
      envVarCount: 0
    };

    this.setEnvVar = this.setEnvVar.bind(this);
    this.deleteEnvVar = this.deleteEnvVar.bind(this);
  }

  render() {
    return (
      <div style={this.styles.root}>
        <h4>Environment variables</h4>
        <FlatButton
          label="Add an environment variable"
          primary={true}
          icon={<ContentAdd />}
          onClick={() => this.addEnvVar()}
        />
        {Object.keys(this.state.envVars).map(k => (
          <EnvVar
            key={k}
            id={k}
            setEnvVar={this.setEnvVar}
            deleteEnvVar={this.deleteEnvVar}
          />
        ))}
      </div>
    );
  }

  setEnvVar(id, envVar) {
    let envVars = { ...this.state.envVars, [id]: envVar };
    this.setState({ envVars });
    this.bubbleSpecs(envVars);
  }

  deleteEnvVar(id) {
    this.setState({ envVars: omit(this.state.envVars, [id]) });
  }

  bubbleSpecs(envVars) {
    let evs = [];
    Object.keys(envVars).forEach(k => {
      const ev = envVars[k];
      evs.push({ name: ev.name, value: ev.value });
    });
    this.props.setEnvVars(evs);
  }

  addEnvVar() {
    const id = this.state.envVarCount;
    this.setState({
      envVarCount: id + 1,
      envVars: { ...this.state.envVars, [id]: {} }
    });
  }

  styles = {
    card: {
      boxShadow: ""
    },
    root: {
      marginTop: "20px"
    }
  };
}

EnvVarCreator.propTypes = {
  setEnvVars: PropTypes.func.isRequired
};

export default EnvVarCreator;
