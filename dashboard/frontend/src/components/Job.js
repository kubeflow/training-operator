import React, { Component } from "react";
import JobDetail from "./JobDetail.js";
import ReplicaSpec from "./ReplicaSpec.js";
import { Card, CardText } from "material-ui/Card";
import { getTFJobService } from "../services";

class Job extends Component {
  constructor(props) {
    super(props);
    this.state = {
      tfJob: null,
      tbService: null,
      pods: []
    };
  }

  divStyle = { marginBottom: "10px" };

  componentWillReceiveProps(nextProps) {
    this.displayJob(nextProps);
  }

  componentDidMount() {
    this.displayJob(this.props);
  }

  render() {
    let job = this.state.tfJob;

    if (job) {
      const tfReplicaSpecs = this.renderTfReplicaSpecs(job);
      return (
        <div>
          <div style={this.divStyle}>
            <JobDetail tfjob={job} />
          </div>
          {tfReplicaSpecs}
        </div>
      );
    }
    return (
      <Card>
        <CardText> There are no TFJobs to display </CardText>
      </Card>
    );
  }

  displayJob(props) {
    let job = this.getCurrentJob(props);
    if (job) {
      getTFJobService(job.metadata.namespace, job.metadata.name)
        .then(b => {
          this.setState({
            tfJob: b.tfJob,
            tbService: b.tbService,
            pods: b.pods
          });
        })
        .catch(console.error);
    }
  }

  getCurrentJob(props) {
    if (!props.jobs || props.jobs.length === 0) {
      return null;
    }

    if (!props.match.params.name || !props.match.params.namespace) {
      return props.jobs[0];
    }
    const matches = props.jobs.filter(
      j =>
        j.metadata.name === props.match.params.name &&
        j.metadata.namespace === props.match.params.namespace
    );
    return matches[0];
  }

  renderTfReplicaSpecs(job) {
    let tfReplicaSpecs = [];
    let i = 0;
    for(let replicaType in job.spec.tfReplicaSpecs) {
      i++;
      let spec = job.spec.tfReplicaSpecs[replicaType];
      let pods = this.state.pods.filter( p =>
          p.metadata.labels["tf-replica-type"] &&
          p.metadata.labels["tf-replica-type"] === replicaType.toLowerCase()
      );
      tfReplicaSpecs.push(
        <div style={this.divStyle} key={i}>
          <ReplicaSpec replicaType={replicaType} spec={spec} pods={pods} />
        </div>
      );
    }

    return tfReplicaSpecs;
  }
}

export default Job;
