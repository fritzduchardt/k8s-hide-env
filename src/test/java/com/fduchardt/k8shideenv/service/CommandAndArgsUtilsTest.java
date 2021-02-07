package com.fduchardt.k8shideenv.service;

import org.junit.jupiter.api.*;
import org.junit.jupiter.params.*;
import org.junit.jupiter.params.provider.*;

import java.util.*;
import java.util.stream.*;

import static org.junit.jupiter.api.Assertions.*;
import static org.junit.jupiter.params.provider.Arguments.*;

class CommandAndArgsUtilsTest {

    private CommandAndArgsUtils underTest = new CommandAndArgsUtils();

    @Test
    public void errorOnEmpty() {
        assertThrows(RuntimeException.class, () -> {
            underTest.reconcile(List.of(), List.of());
        });
    }

    @ParameterizedTest
    @MethodSource("commandAndArgsProvider")
    public void happyPath(List<String> commands, List<String> args, String expected) {
        String actual = underTest.reconcile(commands, args);
        assertEquals(expected, actual);
    }

    static Stream<Arguments> commandAndArgsProvider() {
        return Stream.of(
            arguments(List.of("echo", "hi"), List.of(""), "echo hi"),
            arguments(List.of("sh", "-c", "echo hi"), List.of(""), "echo hi"),
            arguments(List.of("sh", "-c"), List.of("echo hi"), "echo hi"),
            arguments(List.of(), List.of("echo", "hi"), "echo hi"),
            arguments(List.of(), List.of("sh", "-c", "echo hi"), "echo hi"),
            arguments(List.of("sh", "-c"), List.of("sh", "-c", "echo hi"), "echo hi"),
            arguments(List.of(), List.of("sh", "startup.sh"), "sh startup.sh"),
            arguments(List.of("sh", "startup.sh"), List.of(), "sh startup.sh")
        );
    }
}