import React from 'react';
import InfoEntry from './InfoEntry';
import { Card, CardText, CardHeader } from 'material-ui/Card';
import PodList from './PodList';

const ReplicaSpec = ({ spec, status, pods }) => {
    return (
        <Card>
            <CardHeader title={spec.tfReplicaType} textStyle={{ fontWeight: "bold" }} />
            <CardText>
                <InfoEntry name="Replicas" value={spec.replicas} />
                <InfoEntry name="Image" value={spec.template.spec.containers[0].image} />
                <InfoEntry name="State" value={status.state} />
                <PodList pods={pods} />
            </CardText>
        </Card>
    );
}

export default ReplicaSpec;