import React, { Component } from 'react';
import {
    BrowserRouter as Router,
    Route,
    Link,
    Switch,
    withRouter
} from 'react-router-dom'
import './App.css';

import JobList from './JobList'
import Job from './Job'
import CreateJob from './CreateJob'
import AppBar from './AppBar'
import { getTfJobListService } from '../services'

class Home extends Component {

    constructor(props) {
        super(props);
        this.state = {
            tfJobs: []
        }
    }

    componentDidMount() {
        this.fetch();
        // setInterval(_ => this.fetch(), 10000);
    }

    fetch() {
        getTfJobListService()
            .then(b => {
                this.setState({ tfJobs: b.items });
            })
            .catch(console.log);
    }

    render() {
        let job = withRouter(<Job jobs={this.state.tfJobs} />);
        return (
            <div>
                <AppBar />
                <div id="main" style={this.styles.mainStyle} >
                    <div style={this.styles.list}>
                        <JobList jobs={this.state.tfJobs} />
                    </div>
                    <div style={this.styles.content}>
                        <Switch>
                            <Route path="/new" component={CreateJob} />
                            <Route path="/:namespace/:name" render={props => <Job jobs={this.state.tfJobs} {...props} />} />
                            <Route path="/" render={props => <Job jobs={this.state.tfJobs} {...props} />} />
                        </Switch>
                    </div>
                </div>
            </div>
        );
    }

    styles = {
        mainStyle: {
            minHeight: "800px",
            margin: 0,
            padding: 0,
            display: "flex",
            flexFlow: "row"
        },
        content: {
            margin: "4px",
            padding: "5px",
            flex: "3 1 80%",
            order: 2
        },
        list: {
            margin: "4px",
            padding: "5px",
            flex: "1 6 20%",
            order: 1
        }
    }
}

export default Home;
