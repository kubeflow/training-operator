import React from 'react';
import JobSummary from './JobSummary'

let jobsStyle = {
  display: "flex",
  flexDirection: "column"
}

const JobList = ({jobs}) => { 
    return (
      <div style={jobsStyle}>
        {jobs.map((v, k) => <JobSummary key={k} job={v} />)}
      </div>
    );
}

export default JobList;
