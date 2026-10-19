use gtk4::{
    Application, ApplicationWindow, Label,
    gio::prelude::{ApplicationExt, ApplicationExtManual},
    glib,
    prelude::GtkWindowExt,
};

const APP_ID: &str = "org.efournierrobert.cake-desktop";

fn main() -> glib::ExitCode {
    let app = Application::builder().application_id(APP_ID).build();

    app.connect_activate(build_ui);

    app.run()
}

fn build_ui(app: &Application) {
    let label = Label::builder()
        .label("Hello world!")
        .margin_top(12)
        .margin_bottom(12)
        .margin_start(12)
        .margin_end(12)
        .build();

    let window = ApplicationWindow::builder()
        .application(app)
        .title("CAKE Desktop")
        .child(&label)
        .build();

    window.present()
}
