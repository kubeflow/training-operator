import React from 'react';
import { Link } from 'react-router-dom'
import { Card, CardActions, CardHeader, CardMedia, CardTitle, CardText } from 'material-ui/Card';

const styles = {
  jobSummary: {
    textAlign: "left",
    display: "flex"
  },
  indicator: {
    color: "red",
    marginBottom: 0,
    marginLeft: 5
  },
  description: {
    paddingRight: 0
  },
  details: {
    display: "flex",
    flexDirection: "column",
    marginLeft: "4px"
  }
}

const JobSummary = ({ job }) => {
  return (
    <Card>
      <CardText >
        <div style={styles.jobSummary} className="JobSummary Card">
          <div>
            <p style={styles.indicator} >&#9673;</p>
          </div>
          <div style={styles.details}>
            <Link to={`/${job.metadata.uid}`}>{job.metadata.name}</Link>
            <div style={styles.description}>
              {getReplicasSummary(job)}
            </div>
          </div>
        </div>
      </CardText>
    </Card>
  )
}

function getReplicasSummary(job) {

  const descs = job.spec.replicaSpecs.reduce((acc, r) => {
    acc.push(`${r.tfReplicaType.toLowerCase()}: ${r.replicas}`);
    return acc;
  }, []);
  return descs.join()
}

export default JobSummary;
