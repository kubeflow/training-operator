export function getHost() {
  const prefix = "/tfjobs";
  const pLen = prefix.length;

  const fullPath = window.location.pathname;
  if (fullPath.indexOf(prefix) != -1) {
    const keepLen = fullPath.indexOf(prefix) + pLen;
    return fullPath.substr(0, keepLen);
  } else {
    return "";
  }
}
