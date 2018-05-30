import React from "react";
import PropTypes from "prop-types";
import { Link } from "react-router-dom";
import { Card, CardText } from "material-ui/Card";

const JobSummary = ({ job }) => {
  const styles = {
    jobSummary: {
      textAlign: "left",
      display: "flex"
    },
    indicator: {
      color: getStatusColor(job),
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
  };

  return (
    <Card>
      <CardText>
        <div style={styles.jobSummary} className="JobSummary Card">
          <div>
            <p style={styles.indicator}>&#9673;</p>
          </div>
          <div style={styles.details}>
            <Link to={`/${job.metadata.namespace}/${job.metadata.name}`}>
              {job.metadata.name}
            </Link>
            <div style={styles.description}>{getReplicasSummary(job)}</div>
          </div>
        </div>
      </CardText>
    </Card>
  );
};

function getReplicasSummary(job) {
  let descriptions = []
  for(let replicaType in job.spec.tfReplicaSpecs){
    descriptions.push(`${replicaType.toLowerCase()}: ${job.spec.tfReplicaSpecs[replicaType].replicas}`)
  } 
  return descriptions.join()
}

function getStatusColor(job) {
  if(job.status.active > 0) {
    return "orange";
  }
  return "green"
}

JobSummary.propTypes = {
  job: PropTypes.shape({
    spec: PropTypes.object.isRequired,
    metadata: PropTypes.object.isRequired,
    status: PropTypes.object.isRequired
  }).isRequired
};

export default JobSummary;
