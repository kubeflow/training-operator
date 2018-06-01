import React from "react";
import PropTypes from "prop-types";
import InfoEntry from "./InfoEntry";
import { Card, CardText } from "material-ui/Card";

const JobDetail = ({ tfjob }) => {
  let status = "Unknown";
  if(tfjob.status.conditions && tfjob.status.conditions.length > 0) {
    status = tfjob.status.conditions[tfjob.status.conditions.length - 1].reason
  }

  return (
    <div>
      <Card>
        <CardText>
          <div>
            <InfoEntry name="Name" value={tfjob.metadata.name} />
            <InfoEntry name="Namespace" value={tfjob.metadata.namespace} />
            <InfoEntry
              name="Created on"
              value={tfjob.metadata.creationTimestamp}
            />
            {/* <InfoEntry name="Runtime Id" value={tfjob.spec.RuntimeId} /> */}
            <InfoEntry name="Status" value={status} />
          </div>
        </CardText>
      </Card>
    </div>
  );
}

JobDetail.propTypes = {
  tfjob: PropTypes.shape({
    metadata: PropTypes.object.isRequired,
    spec: PropTypes.object.isRequired
  }).isRequired
};

export default JobDetail;
