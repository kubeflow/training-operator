import React from "react";
import PropTypes from "prop-types";
import JobSummary from "./JobSummary";

const jobsStyle = {
  display: "flex",
  flexDirection: "column"
};

const JobList = ({ jobs }) => {
  return (
    <div style={jobsStyle}>
      {jobs.map((v, k) => <JobSummary key={k} job={v} />)}
    </div>
  );
};

JobList.propTypes = {
  jobs: PropTypes.arrayOf(PropTypes.object).isRequired
};

export default JobList;
