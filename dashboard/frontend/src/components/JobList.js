import React from 'react';
import JobSummary from './JobSummary'

let jobsStyle = {
  display: "flex",
  flexDirection: "column"
}

const JobList = ({jobs}) => { 
    let jobSummaries = []
    for (let i = 0; i < jobs.length; i++) {
      jobSummaries.push(<JobSummary key={i} job={jobs[i]} />)
    }
    
    return (
      <div style={jobsStyle}>
        {jobSummaries}
      </div>
    );
}

export default JobList;
