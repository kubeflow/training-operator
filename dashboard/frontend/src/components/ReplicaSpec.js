import React from 'react';
import InfoEntry from './InfoEntry'
import { Card, CardText, CardHeader } from 'material-ui/Card';

const ReplicaSpec = ({ spec, status }) => {
    return (
        <Card>
            <CardHeader title={spec.tfReplicaType} textStyle={{fontWeight: "bold"}}/>
            <CardText>
                <div>
                    <InfoEntry name="Replicas" value={spec.replicas} />
                    <InfoEntry name="Image" value={spec.template.spec.containers[0].image} />
                    <InfoEntry name="State" value={status.state} />
                </div>
            </CardText>
        </Card>
    )
}

export default ReplicaSpec