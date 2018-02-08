import React, { Component } from "react";
import PropTypes from "prop-types";
import TextField from "material-ui/TextField";

const errorMessage = "This field is required";

class RequiredTextField extends Component {
  constructor(props) {
    super(props);
    this.state = {
      errorText: this.props.value ? "" : errorMessage
    };
    this.handleChange = this.handleChange.bind(this);
  }

  render() {
    return (
      <div>
        <TextField
          {...this.props}
          errorText={this.state.errorText}
          name={this.props.name}
          onChange={this.handleChange}
        />
      </div>
    );
  }

  handleChange(event) {
    if (!event.target.value) {
      this.setState({ errorText: errorMessage });
    } else {
      this.setState({ errorText: "" });
    }
    this.props.onChange(event);
  }
}

RequiredTextField.propTypes = {
  onChange: PropTypes.func.isRequired,
  name: PropTypes.string.isRequired,
  value: PropTypes.string
};

export default RequiredTextField;
