import React, { Component } from 'react';
import InfoEntry from './InfoEntry'
import JobDetail from './JobDetail.js'
import ReplicaSpec from './ReplicaSpec.js'
import TensorBoard from './TensorBoard.js'
import { Card, CardTitle, CardText } from 'material-ui/Card';
import Divider from 'material-ui/Divider';
import { getTfJobService } from '../services'

const jobStyle = {
  backgroundColor: "white"
}

class Job extends Component {
  constructor(props) {
    super(props);
    this.state = {
      tfJob: null,
      tbService: null,
      pods: []
    }
  }
  divStyle = { marginBottom: "10px" }

  componentWillReceiveProps(nextProps) {
    this.displayJob(nextProps);
  }

  componentDidMount() {
    this.displayJob(this.props);
  }

  render() {
    let job = this.state.tfJob;

    if (job) {
      const replicaSpecs = this.renderReplicaSpecs(job);
      return (
        <div>
          <div style={this.divStyle}>
            <JobDetail tfjob={job} />
          </div>
          <div style={this.divStyle}>
            <TensorBoard service={this.state.tbService} />
          </div>
          {replicaSpecs}
        </div>
      );
    }
    return (
      <Card>
        <CardText> There are no TfJobs to display </CardText>
      </Card>
    );
  }

  displayJob(props) {
    let job = this.getCurrentJob(props);
    if (job) {
      getTfJobService(job.metadata.namespace, job.metadata.name)
        .then(b => {
          this.setState({ tfJob: b.tfJob, tbService: b.tbService, pods: b.pods });
        })
        .catch(console.log);
    }
  }

  getCurrentJob(props) {
    if (!props.jobs || props.jobs.length == 0) {
      return null;
    }

    if (!props.match.params.name || !props.match.params.namespace) {
      return props.jobs[0];
    }
    const matches = props.jobs.filter(j => j.metadata.name == props.match.params.name && j.metadata.namespace == props.match.params.namespace)
    return matches[0]
  }

  renderReplicaSpecs(job) {
    let replicaSpecs = []
    for (let i = 0; i < job.spec.replicaSpecs.length; i++) {
      let spec = job.spec.replicaSpecs[i];
      let status = {
        state: "Unknown"
      }
      if (job.status.replicaStatuses) {
        const m = job.status.replicaStatuses.filter(s => s.tf_replica_type == spec.tfReplicaType)
        if (m.length > 0) {
          status = m[0];
        }
      }

      let pods = this.state.pods.filter(p => p.metadata.labels.job_type && p.metadata.labels.job_type == spec.tfReplicaType);
      replicaSpecs.push(
        <div style={this.divStyle} key={i}>
          <ReplicaSpec spec={spec} status={status} pods={pods}/>
        </div>
      );
    }
    return replicaSpecs;
  }

}

export default Job;
