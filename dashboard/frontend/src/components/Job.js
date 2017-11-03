import React, { Component } from 'react';
import InfoEntry from './InfoEntry'
import JobDetail from './JobDetail.js'
import ReplicaSpec from './ReplicaSpec.js'
import TensorBoard from './TensorBoard.js'
import { Card, CardTitle, CardText } from 'material-ui/Card';
import Divider from 'material-ui/Divider';
const jobStyle = {
  backgroundColor: "white"
}


class Job extends Component {
  constructor(props) {
    super(props);
    this.state = {
      tfJob: null,
      tbService: null,
      storageDetail: null
    }
  }

  divStyle = { marginBottom: "10px" }

  componentWillReceiveProps(nextProps) {
    let job = nextProps.job;
    if (job) {
      fetch(`http://localhost:8080/api/tfjob/${job.metadata.namespace}/${job.metadata.name}`)
        .then(r => r.json())
        .then(b => {
          this.setState({ tfJob: b.tfJob, tbService: b.tbService });
        })
        .catch(console.log);
    }
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
            <TensorBoard service={this.state.tbService}  />
          </div>
          {replicaSpecs}
        </div>
      );
    }
    return (
      <Card />
    );
  }

  renderReplicaSpecs(job) {
    let replicaSpecs = []
    for (let i = 0; i < job.spec.replicaSpecs.length; i++) {
      let spec = job.spec.replicaSpecs[i];
      let status = job.status.replicaStatuses.filter(s => s.tf_replica_type == spec.tfReplicaType)[0];
      replicaSpecs.push(
        <div style={this.divStyle} key={i}>
          <ReplicaSpec spec={spec} status={status} />
        </div>
      );
    }
    return replicaSpecs;
  }

}

export default Job;
