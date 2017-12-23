import React from 'react';
import InfoEntry from './InfoEntry';
import { Card, CardText } from 'material-ui/Card';

const JobDetail = ({ tfjob }) => (
    <div>
        <Card>
            <CardText>
                <div>
                    <InfoEntry name="Name" value={tfjob.metadata.name} />
                    <InfoEntry name="Namespace" value={tfjob.metadata.namespace} />
                    <InfoEntry name="Created on" value={tfjob.metadata.creationTimestamp} />
                    <InfoEntry name="Runtime Id" value={tfjob.spec.RuntimeId} />
                </div>
            </CardText>
        </Card>
    </div>
);

export default JobDetail;
