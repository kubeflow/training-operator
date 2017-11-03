import React, { Component } from 'react';
import JobList from './JobList'
import Job from './Job'
import CreateJob from './CreateJob'
import {
    BrowserRouter as Router,
    Route,
    Link,
    Switch,
    withRouter
} from 'react-router-dom'
import './App.css';
import AppBar from 'material-ui/AppBar';
class Home extends Component {

    constructor(props) {
        super(props);
        this.state = {
            tfJobs: []
        }
    }

    componentDidMount() {
        setInterval(_ => this.fetchTfJobs(), 10000)
    }

    fetchTfJobs() {
        fetch("http://localhost:8080/api/tfjob")
            .then(r => r.json())
            .then(b => {
                this.setState({ tfJobs: b.items })
            })
            .catch(console.log);
    }

    render() {
        return (
            <div>
                <AppBar
                    title={this.getTitle()}
                    iconClassNameRight="muidocs-icon-navigation-expand-more"
                />
                <div id="main" style={this.styles.mainStyle} >
                    <div style={this.styles.list}>
                        <JobList jobs={this.state.tfJobs} />
                    </div>
                    <div style={this.styles.content}>
                        <Switch>
                            <Route path="/new" component={CreateJob} />
                            <Route path="/:name" render={(props) => {
                                let job = this.state.tfJobs.filter(j => j.metadata.name == props.match.params.name)
                                return (<Job job={job[0]} />)
                            }
                            } />
                            <Route exact path="/" render={() =>
                                <Job job={this.state.tfJobs[0]} />
                            } />
                        </Switch>
                    </div>
                </div>
            </div>
        );
    }

    getTitle() {
        const path = this.props.location.pathname;
        switch (path) {
            case "/new":
                return "New Training";
            default:
                return path.replace('/', '')
        }
        return this.props.location.pathname
    }

    styles = {
        mainStyle: {
            minHeight: "800px",
            margin: 0,
            padding: 0,
            display: "flex",
            flexFlow: "row"
        },
        header: {
            backgroundColor: "#f57c00",
            height: 56,
            color: "white",
            fontSize: 18,
            textAlign: "center",
            padding: 15
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

export default withRouter(Home);
