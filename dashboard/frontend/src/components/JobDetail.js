import React, { Component } from 'react';
import InfoEntry from './InfoEntry'
import ReplicaSpec from './ReplicaSpec.js'
import TensorBoard from './TensorBoard.js'
import { Card, CardTitle, CardText } from 'material-ui/Card';
import Divider from 'material-ui/Divider';

const JobDetail = ({ tfjob }) => (
    <div>
        <Card>
            <CardTitle title={tfjob.metadata.name} />
            <CardText>
                <div>
                    <InfoEntry name="Namespace" value={tfjob.metadata.namespace} />
                    <InfoEntry name="Created on" value={tfjob.metadata.creationTimestamp} />
                    <InfoEntry name="Runtime Id" value={tfjob.spec.RuntimeId} />
                </div>
            </CardText>
        </Card>
    </div>
);

export default JobDetail;
