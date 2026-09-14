use crate::components::{AcceptInvitation, Dashboard, HouseDetail, Login, Register};
use dioxus::prelude::*;

#[derive(Clone, Routable, Debug, PartialEq)]
pub enum Route {
    #[route("/")]
    Login {},
    #[route("/register")]
    Register {},
    #[route("/accept-invitation?:token")]
    AcceptInvitation { token: Option<String> },
    #[route("/dashboard")]
    Dashboard {},
    #[route("/house/:id")]
    HouseDetail { id: String },
}
