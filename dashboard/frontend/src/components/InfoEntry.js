import React, { Component } from "react";
import PropTypes from "prop-types";

class InfoEntry extends Component {
  render() {
    const styles = {
      root: {
        display: "flex",
        paddingBottom: "8px"
      },
      label: {
        fontWeight: "bold"
      },
      value: {
        marginLeft: "4px"
      }
    };

    return (
      <div style={styles.root}>
        <div style={styles.label}>{this.props.name + ":"}</div>
        <div style={styles.value}>
          {this.props.linkTo ? (
            <a href={this.props.linkTo} target="_blank">
              {this.props.value}
            </a>
          ) : (
            this.props.value
          )}
        </div>
      </div>
    );
  }
}

InfoEntry.propTypes = {
  linkTo: PropTypes.string.isRequired,
  name: PropTypes.string.isRequired,
  value: PropTypes.string.isRequired
};

export default InfoEntry;
