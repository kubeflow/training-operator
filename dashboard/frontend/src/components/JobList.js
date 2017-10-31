import React, { Component } from 'react';
import { Row, Col, Grid } from 'react-bootstrap';
import JobSummary from './JobSummary'

class JobList extends Component {

  render() {
    let jobsStyle = {
    }
    return (
      <div className="JobList CardBox" style={jobsStyle}>
        <JobSummary />
        <JobSummary />
        <JobSummary />
        <JobSummary />
      </div>
    );
  }
}

export default JobList;
