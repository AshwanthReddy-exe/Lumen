plugins {
    kotlin("jvm")
    application
}

dependencies {
    implementation(project(":core:space"))
}

application {
    mainClass = "dev.lumen.scenario.MainKt"
}
