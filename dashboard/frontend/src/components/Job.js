import React, { Component } from 'react';
import { Row, Col, Grid } from 'react-bootstrap';
import FormEntry from './FormEntry'

class Job extends Component {

  render() {

    let jobStyle = {
      backgroundColor: "white"
    }

    return (
      <div className="Job CardBox" style={jobStyle} >
       <FormEntry name="Name" value="distributed-tf"/>
       <FormEntry name="Runtime Id" value="teva"/>
       <FormEntry name="Image" value="wbuchwalter/distributed-tfjob"/>
      </div>
    );
  }
}

export default Job;
