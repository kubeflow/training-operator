import React, { Component } from 'react';
import MuiThemeProvider from 'material-ui/styles/MuiThemeProvider';
import getMuiTheme from 'material-ui/styles/getMuiTheme';
import FlatButton from 'material-ui/FlatButton'
import {
  BrowserRouter as Router,
  Link
} from 'react-router-dom'
import { orange700, orange400 } from 'material-ui/styles/colors';
import muiThemeable from 'material-ui/styles/muiThemeable';
import ContentAdd from 'material-ui/svg-icons/content/add';

import './App.css';
import Home from './Home.js'

let headerStyle = {
  display: "flex",
  backgroundColor: "white",
  height: "56px",
  padding: "12px",
  color: "#f57c00",
  justifyContent: "space-between"
};

let brandingStyle = {
  fontSize: "1.5em",
  marginTop: "3px",
  textAlign: "left"
}

const muiTheme = getMuiTheme({ 
  appBar: {
    height: 56,
    color: orange700
  },
  raisedButton: {
    primaryColor: orange700
  },
  flatButton: {
    primaryTextColor: orange700
  },
  toggle: {
    thumbOnColor: orange700,
    trackOnColor: orange400
  }
});

const App = props => { 
  // console.log(props.muiTheme)
  return(
  <Router>
    <MuiThemeProvider muiTheme={muiTheme}>
      <div className="App">
        <header className="App-header">
          <link href="https://fonts.googleapis.com/css?family=Roboto" rel="stylesheet" />
          <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/latest/css/bootstrap.min.css" />
          <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet" />
          <div style={headerStyle}>
            <h1 style={brandingStyle}> KubeFlow</h1>
            <FlatButton label="Create" primary={true} icon={<ContentAdd />} containerElement={<Link to="/new" />} />
          </div>
        </header>
        <div>
          <Home />
        </div>
      </div>
    </MuiThemeProvider>
  </Router>
)
};


// export default muiThemeable()(App);
export default App;

