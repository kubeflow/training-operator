import google.auth
import google.auth.transport
import google.auth.transport.requests
import os
print(os.getenv("GOOGLE_APPLICATION_CREDENTIALS", ""))
#credentials, project_id = google.auth.default(scopes=["https://www.googleapis.com/auth/userinfo.email"])
credentials, project_id = google.auth.default()
request = google.auth.transport.requests.Request()
credentials.refresh(request)

