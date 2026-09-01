use crate::components::{Dashboard, HouseDetail, Login, Register};
use dioxus::prelude::*;

#[derive(Clone, Routable, Debug, PartialEq)]
pub enum Route {
    #[route("/")]
    Login {},
    #[route("/register")]
    Register {},
    #[route("/dashboard")]
    Dashboard {},
    #[route("/house/:id")]
    HouseDetail { id: String },
}
