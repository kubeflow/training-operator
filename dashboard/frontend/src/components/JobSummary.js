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
  const descs = job.spec.replicaSpecs.reduce((acc, r) => {
    acc.push(`${r.tfReplicaType.toLowerCase()}: ${r.replicas}`);
    return acc;
  }, []);
  return descs.join();
}

function getStatusColor(job) {
  let status = job.status.state;
  switch (status.toLowerCase()) {
    case "succeeded":
      return "green";
    default:
      return "orange";
  }
}

JobSummary.propTypes = {
  job: PropTypes.shape({
    spec: PropTypes.object.isRequired,
    metadata: PropTypes.object.isRequired,
    status: PropTypes.object.isRequired
  }).isRequired
};

export default JobSummary;
