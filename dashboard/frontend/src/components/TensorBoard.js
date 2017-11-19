import React from 'react';
import InfoEntry from './InfoEntry';
import { Card, CardHeader, CardText } from 'material-ui/Card';

const TensorBoard = ({ service }) => {
    return (
        <Card>
            <CardHeader title="TensorBoard" textStyle={{ fontWeight: "bold" }} />
            <CardText>
                {service ?
                    <div>
                        <InfoEntry name="Cluster IP" value={service.spec.clusterIP} linkTo={"http://" + service.spec.clusterIP} />
                        {service.status.loadBalancer.ingress &&
                            <InfoEntry name="External endpoints" value={service.status.loadBalancer.ingress[0].ip} linkTo={"http://" + service.status.loadBalancer.ingress[0].ip} />
                        }
                    </div> :
                    "TensorBoard was not configured for this TfJob or the service was deleted."
                }
            </CardText>
        </Card>
    );
}

export default TensorBoard;