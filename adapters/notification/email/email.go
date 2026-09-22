package email

// This package contains an email delivery adapter.
//
// It translates the generic notification representation into the
// API/protocol required by the selected email provider.
//
// Under the primary DispatchAI integration architecture, actual
// notification delivery should normally remain inside the Host Project.
//
// This adapter exists only when DispatchAI itself is intentionally
// responsible for notification delivery.
