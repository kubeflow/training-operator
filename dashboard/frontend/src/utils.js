export function getHost() {
  const prefix = "/tfjobs";
  const pLen = prefix.length;

  const fullPath = window.location.pathname;
  const keepLen = fullPath.indexOf(prefix) + pLen;
  return fullPath.substr(0, keepLen);
}
