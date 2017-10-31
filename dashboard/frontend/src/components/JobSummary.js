import React, { Component } from 'react';
import { Row, Col, Grid } from 'react-bootstrap';

class JobSummary extends Component {

  render() {
    let styles = {
      indicatorStyle: {
        color: "red",
        marginBottom: 0,
        marginLeft: 5
      },
      jobSummaryStyle: {
        textAlign: "left",
      },
      descriptionStyle: {
        paddingRight: 0
      }
    }

    return (
      <div style={styles.jobSummaryStyle} class="JobSummary Card">
         <Row>
          <Col md={2}>
            <p style={styles.indicatorStyle} >&#9673;</p>
          </Col>
          <Col md={10}>
            <a>distributed-tf</a>
          </Col>
        </Row>
        <Row>
          <Col style={styles.descriptionStyle} md={10} mdOffset={2}>
            Worker(s): 2, PS: 2
          </Col>
        </Row>
      </div>
    );
  }
}

export default JobSummary;
