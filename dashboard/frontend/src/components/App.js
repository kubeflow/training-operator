import React, { Component } from 'react';
import { Row, Col, Grid } from 'react-bootstrap';
import '../assets/react-toolbox/theme.css';
import theme from '../assets/react-toolbox/theme.js';
import ThemeProvider from 'react-toolbox/lib/ThemeProvider';
import Button from 'react-toolbox/lib/button/Button';
import JobList from './JobList'
import Job from './Job'

import './App.css';

class App extends Component {

  render() {
    let noMargin = {
      margin: 0
    }

    let sectionTitleStyle = {
      backgroundColor: "#ef6c00",
      height: 56,
      color: "white",
      fontSize: 18,
      textAlign: "center",
    }

    let mainContent = {
      marginTop: 15
    }
    
    return (
      <ThemeProvider theme={theme}>
      <div className="App" style={noMargin}>
        <header className="App-header">
          <link href="https://fonts.googleapis.com/css?family=Roboto" rel="stylesheet" />
          <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/latest/css/bootstrap.min.css" />
          <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet" />
          <Grid>
            <Row >
              <Col md={1}>
                <h1 className="App-title">MLKube.io</h1>
              </Col>
              <Col md={2} mdOffset={8}>
                <Button icon='add' label='Create' style={{color: "#ef6c00"}} flat primary />
              </Col>
            </Row>
          </Grid>
        </header>
        <Grid fluid>
          <Row className="section-title" style={sectionTitleStyle}>
            <Col xs={12} style={{marginTop: 15}}>
              TfJob Overview
            </Col>
          </Row>
          <Row className="show-grid" style={mainContent}>
            <Col xs={12} md={2}>
              <JobList />
            </Col>
            <Col xs={12} mdOffset={1} md={8}>
              <Job />
            </Col>
          </Row>
        </Grid>
      </div>
      </ThemeProvider>
    );
  }
}

export default App;
