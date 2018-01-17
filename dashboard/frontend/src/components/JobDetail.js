import React from "react";
import PropTypes from "prop-types";
import InfoEntry from "./InfoEntry";
import { Card, CardText } from "material-ui/Card";

const JobDetail = ({ tfjob }) => (
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
          <InfoEntry name="Runtime Id" value={tfjob.spec.RuntimeId} />
        </div>
      </CardText>
    </Card>
  </div>
);

JobDetail.propTypes = {
  tfjob: PropTypes.shape({
    metadata: PropTypes.object.isRequired,
    spec: PropTypes.object.isRequired
  }).isRequired
};

export default JobDetail;
