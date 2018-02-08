import React from "react";
import PropTypes from "prop-types";
import TextField from "material-ui/TextField";
import FlatButton from "material-ui/FlatButton";
import ActionDelete from "material-ui/svg-icons/action/delete";

class EnvVar extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      name: "",
      value: ""
    };
    this.handleDelete = this.handleDelete.bind(this);
    this.handleInputChange = this.handleInputChange.bind(this);
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
      <div style={this.styles.content}>
        <TextField
          floatingLabelText="Name"
          value={this.state.name}
          name="name"
          onChange={this.handleInputChange}
          style={this.styles.element}
        />
        <TextField
          floatingLabelText="Value"
          value={this.state.value}
          name="value"
          onChange={this.handleInputChange}
          style={this.styles.element}
        />
        <FlatButton
          style={this.styles.deleteIcon}
          icon={<ActionDelete />}
          onClick={this.handleDelete}
        />
      </div>
    );
  }

  styles = {
    content: {
      display: "flex",
      flexDirection: "row"
    },
    element: {
      marginRight: "36px"
    },
    deleteIcon: {
      marginTop: "30px"
    }
  };

  bubbleSpec(state) {
    this.props.setEnvVar(this.props.id, state);
  }

  handleDelete() {
    this.props.deleteEnvVar(this.props.id);
  }
}

EnvVar.propTypes = {
  id: PropTypes.string.isRequired,
  setEnvVar: PropTypes.func.isRequired,
  deleteEnvVar: PropTypes.func.isRequired
};

export default EnvVar;
