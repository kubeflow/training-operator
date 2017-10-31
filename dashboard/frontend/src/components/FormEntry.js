import React, { Component } from 'react';
import { Row, Col, Grid } from 'react-bootstrap';

class FormEntry extends Component {

  render() {
    let formNameStyle = {
      textAlign: "right",
      fontWeight: "bold"
    }
    let entryStyle = {
      textAlign: "left"
    }
    return (
      <Grid className="FormEntry" >
        <Row>
          <Col md={2} style={formNameStyle}>
            {this.props.name + ":"}
          </Col>
          <Col md={3} style={entryStyle}>
            {this.props.value}
          </Col>
        </Row>
      </Grid>
    );
  }
}

export default FormEntry;
