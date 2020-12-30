package com.fduchardt.k8shideenv.service;

import org.junit.jupiter.api.*;
import org.junit.jupiter.api.extension.*;
import org.mockito.junit.jupiter.*;
import org.springframework.context.annotation.*;

import java.io.*;
import java.net.*;
import java.nio.file.*;

@ExtendWith(MockitoExtension.class)
@Profile("test")
class K8sHideEnvServiceTest {

    K8sHideEnvService underTest = new K8sHideEnvService();

    @Test
    public void test() throws URISyntaxException, IOException {
        String json = underTest.createAdmissionResponse(Files.readString(Paths.get(ClassLoader.getSystemResource("admission-request.yml").toURI())));
        System.out.println(json);
    }

}