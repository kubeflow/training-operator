import React, { Component } from 'react';
import TextField from 'material-ui/TextField';

const errorMessage = "This field is required"

class RequiredTextField extends Component {
  constructor(props){
    super(props)
    this.state = {
      errorText: this.props.value ? "" : errorMessage
    }


    this.handleChange = this.handleChange.bind(this);
  }

  render() {
    return (
      <div>
        {/* <TextField floatingLabelText="Replicas" type="number" min="0" name="workerReplicas" value={this.state.workerReplicas} onChange={this.handleInputChange} /> */}
        <TextField {...this.props} errorText={this.state.errorText} onChange={this.handleChange}/>
      </div>
    )
  }

  handleChange(event){
    console.log(event.target.value)
    if(!event.target.value) {
      this.setState({errorText: errorMessage})
    } else {
      this.setState({errorText: ""})
      
    }
  }
}

export default RequiredTextField;