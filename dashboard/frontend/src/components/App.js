import React, { Component } from 'react';
import MuiThemeProvider from 'material-ui/styles/MuiThemeProvider';
import FlatButton from 'material-ui/FlatButton'
import Home from './Home.js'
import {
  BrowserRouter as Router
} from 'react-router-dom'
import './App.css';

class App extends Component {

  render() {
    let headerStyle = {
      display: "flex",
      backgroundColor: "white",
      height: "56px",
      padding: "12px",
      color: "#f57c00"
    };
    let brandingStyle = {
      fontSize: "1.5em",
      marginTop: "3px",
      textAlign: "left"
    }

    return (
      <Router>
        <MuiThemeProvider>
          <div className="App">
            <header className="App-header">
              <link href="https://fonts.googleapis.com/css?family=Roboto" rel="stylesheet" />
              <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/latest/css/bootstrap.min.css" />
              <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet" />
              <div style={headerStyle}>
                <h1 style={brandingStyle} > KubeFlow</h1>
                <FlatButton label="Create" />
              </div>
            </header>
            <div>
              <Home />
            </div>
          </div>
        </MuiThemeProvider>
      </Router>
    );
  }
}

export default App;
